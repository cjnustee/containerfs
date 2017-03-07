package cfs

import (
	"bufio"
	"bytes"
	//"encoding/json"
	"fmt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"io"
	dp "ipd.org/containerfs/proto/dp"
	mp "ipd.org/containerfs/proto/mp"
	vp "ipd.org/containerfs/proto/vp"
	"ipd.org/containerfs/utils"
	"os"
	"strconv"
	"sync"
)

const (
	volMgrAddress   = "10.8.65.94:10001"
	metaDataAddress = "10.8.65.94:10002"
)

const (
	O_RDONLY = os.O_RDONLY // 0    00000000000000000000000000000000
	O_WRONLY = os.O_WRONLY // 1    00000000000000000000000000000001
	O_RDWR   = os.O_RDWR   // 2    00000000000000000000000000000010
	O_APPEND = os.O_APPEND // 1024 00000000000000000000010000000000
	O_CREATE = os.O_CREATE // 64   00000000000000000000000001000000
	O_TRUNC  = os.O_TRUNC  // 512  00000000000000000000001000000000
	O_DIRECT = 0x4000
)

const (
	chunkSize = 64 * 1024 * 1024
)

type CFS struct {
	volID string
}

var Wg sync.WaitGroup

func CreateVol(name string, capacity string) int32 {
	//fmt.Println("createVol...")
	// send to volmgr to allcate a new vol
	conn, err := grpc.Dial(volMgrAddress, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("did not connect: %v", err)
	}
	defer conn.Close()
	vc := vp.NewVolMgrClient(conn)
	spaceQuota, _ := strconv.Atoi(capacity)
	pCreateVolReq := &vp.CreateVolReq{
		VolName:    name,
		SpaceQuota: int32(spaceQuota),
	}
	pCreateVolAck, _ := vc.CreateVol(context.Background(), pCreateVolReq)

	fmt.Println(pCreateVolAck)

	// send to metadata to registry a new map
	conn2, err2 := grpc.Dial(metaDataAddress, grpc.WithInsecure())
	if err2 != nil {
		fmt.Printf("did not connect: %v", err2)
		return -1
	}
	defer conn2.Close()
	mc := mp.NewMetaNodeClient(conn2)
	pmCreateNameSpaceReq := &mp.CreateNameSpaceReq{
		VolID: pCreateVolAck.UUID,
	}
	pmCreateNameSpaceAck, _ := mc.CreateNameSpace(context.Background(), pmCreateNameSpaceReq)
	if pmCreateNameSpaceAck.Ret != 0 {
		fmt.Println("CreateNameSpace failed ...")
		return -1
	}

	return 0
}

func GetVolInfo(name string) (int32, *vp.VolInfo) {
	//fmt.Println("GetVolInfo...")
	conn, err := grpc.Dial(volMgrAddress, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("did not connect: %v", err)
	}
	defer conn.Close()
	vc := vp.NewVolMgrClient(conn)
	pGetVolInfoReq := &vp.GetVolInfoReq{
		UUID: name,
	}
	pGetVolInfoAck, _ := vc.GetVolInfo(context.Background(), pGetVolInfoReq)

	if pGetVolInfoAck.Ret != 0 {
		return 1, nil
	}
	return 0, pGetVolInfoAck.VolInfo
}

func OpenFileSystem(UUID string) *CFS {
	cfs := CFS{volID: UUID}
	return &cfs
}

func (cfs *CFS) CreateDir(path string) int32 {
	//fmt.Println("CreateDir...")
	conn, err := grpc.Dial(metaDataAddress, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("did not connect: %v", err)
	}
	defer conn.Close()
	mc := mp.NewMetaNodeClient(conn)
	pCreateDirReq := &mp.CreateDirReq{
		FullPathName: path,
		VolID:        cfs.volID,
	}
	pCreateDirAck, _ := mc.CreateDir(context.Background(), pCreateDirReq)

	return pCreateDirAck.Ret

}
func (cfs *CFS) Stat(path string) (int32, *mp.InodeInfo) {
	conn, err := grpc.Dial(metaDataAddress, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("did not connect: %v", err)
	}
	defer conn.Close()
	mc := mp.NewMetaNodeClient(conn)
	pStatReq := &mp.StatReq{
		FullPathName: path,
		VolID:        cfs.volID,
	}
	pStatAck, _ := mc.Stat(context.Background(), pStatReq)

	return pStatAck.Ret, pStatAck.InodeInfo

}
func (cfs *CFS) List(path string) (int32, []*mp.InodeInfo) {
	conn, err := grpc.Dial(metaDataAddress, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("did not connect: %v", err)
	}
	defer conn.Close()
	mc := mp.NewMetaNodeClient(conn)
	pListReq := &mp.ListReq{
		FullPathName: path,
		VolID:        cfs.volID,
	}
	pListAck, _ := mc.List(context.Background(), pListReq)

	return pListAck.Ret, pListAck.InodeInfos

}
func (cfs *CFS) DeleteDir(path string) int32 {
	conn, err := grpc.Dial(metaDataAddress, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("did not connect: %v", err)
	}
	defer conn.Close()
	mc := mp.NewMetaNodeClient(conn)
	pDeleteDirReq := &mp.DeleteDirReq{
		FullPathName: path,
		VolID:        cfs.volID,
	}
	pDeleteDirAck, _ := mc.DeleteDir(context.Background(), pDeleteDirReq)

	return pDeleteDirAck.Ret
}

func (cfs *CFS) Rename(path1 string, path2 string) int32 {
	conn, err := grpc.Dial(metaDataAddress, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("did not connect: %v", err)
	}
	defer conn.Close()
	mc := mp.NewMetaNodeClient(conn)
	pRenameReq := &mp.RenameReq{
		FullPathName1: path1,
		FullPathName2: path2,
		VolID:         cfs.volID,
	}
	pRenameAck, _ := mc.Rename(context.Background(), pRenameReq)

	return pRenameAck.Ret
}

func (cfs *CFS) OpenFile(path string, flags int) (int32, *CFile) {
	fmt.Println("OpenFile ...")
	fmt.Println(flags)

	var ret int32
	var writer int32 = 0
	var tmpFileSize int64 = 0
	var buf []byte
	cfile := CFile{}

	if flags == O_RDONLY {
		chunkInfos := make([]*mp.ChunkInfo, 0)
		if ret, chunkInfos = cfs.GetFileChunks(path); ret != 0 {
			return ret, nil
		}

		for i := range chunkInfos {
			tmpFileSize += int64(chunkInfos[i].ChunkSize)
		}
		buf = make([]byte, 0, 67108864)
		cfile = CFile{
			Path:       path,
			OpenFlag:   flags,
			cfs:        cfs,
			Writer:     writer,
			fileSize:   tmpFileSize,
			chunks:     chunkInfos,
			lastoffset: 0,
			readBuf:    buf,
		}
	} else if (flags&O_APPEND) != 0 && (flags&O_CREATE) == 0 {

		chunkInfos := make([]*mp.ChunkInfo, 0)

		if ret, chunkInfos = cfs.GetFileChunks(path); ret != 0 {
			return ret, nil
		}

		for i := range chunkInfos {
			tmpFileSize += int64(chunkInfos[i].ChunkSize)
		}
		lastChunk := chunkInfos[len(chunkInfos)-1]

		fmt.Printf("lastChunk:%v\n", lastChunk)

		wChunk := Chunk{
			buffer:    new(bytes.Buffer),
			freeSize:  chunkSize - lastChunk.ChunkSize,
			chunkInfo: lastChunk,
		}

		wChannel := make(chan *Chunk, 16)
		cfile = CFile{
			Path:     path,
			OpenFlag: flags,
			cfs:      cfs,
			Writer:   writer,
			fileSize: tmpFileSize,
			chunks:   chunkInfos,
			wChannel: wChannel,
			wChunk:   wChunk,
		}
		go cfile.flushChannel()

	} else {

		writer = 1
		cfs.DeleteFile(path)
		if ret = cfs.CreateFile(path); ret != 0 {
			return ret, nil
		}

		wChunk := Chunk{freeSize: 0}
		wChannel := make(chan *Chunk, 16)
		cfile = CFile{
			Path:     path,
			OpenFlag: flags,
			cfs:      cfs,
			Writer:   writer,
			fileSize: tmpFileSize,
			wChannel: wChannel,
			wChunk:   wChunk,
		}
		go cfile.flushChannel()
	}
	return 0, &cfile
}

func (cfs *CFS) CreateFile(path string) int32 {
	conn, err := grpc.Dial(metaDataAddress, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("did not connect: %v", err)
	}
	defer conn.Close()
	mc := mp.NewMetaNodeClient(conn)
	pCreateFileReq := &mp.CreateFileReq{
		FullPathName: path,
		VolID:        cfs.volID,
	}
	pCreateFileAck, _ := mc.CreateFile(context.Background(), pCreateFileReq)
	if pCreateFileAck.Ret == 1 {
		fmt.Print("not allowed\n")
		return 1
	}
	if pCreateFileAck.Ret == 2 {
		fmt.Print("no parent path\n")
		return 2
	}
	if pCreateFileAck.Ret == 17 {
		fmt.Print("already exist\n")
		return 17
	}
	return 0
}

func (cfs *CFS) DeleteFile(path string) int32 {

	ret, chunkInfos := cfs.GetFileChunks(path)
	if ret != 0 {
		return ret
	}

	for _, v1 := range chunkInfos {
		for _, v2 := range v1.BlockGroup.BlockInfos {

			addr := utils.Inet_ntoa(v2.DataNodeIP).String() + ":" + strconv.Itoa(int(v2.DataNodePort))
			conn, err := grpc.Dial(addr, grpc.WithInsecure())
			if err != nil {
				fmt.Printf("did not connect: %v", err)
			}

			dc := dp.NewDataNodeClient(conn)

			dpDeleteChunkReq := &dp.DeleteChunkReq{
				ChunkID: v1.ChunkID,
				BlockID: v2.BlockID,
			}
			dpDeleteChunkAck, _ := dc.DeleteChunk(context.Background(), dpDeleteChunkReq)
			if dpDeleteChunkAck.Ret != 0 {
				//return dpDeleteChunkAck.Ret
			}
			conn.Close()
		}
	}

	conn, err := grpc.Dial(metaDataAddress, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("did not connect: %v", err)
	}
	defer conn.Close()
	mc := mp.NewMetaNodeClient(conn)
	mpDeleteFileReq := &mp.DeleteFileReq{
		FullPathName: path,
		VolID:        cfs.volID,
	}
	mpDeleteFileAck, _ := mc.DeleteFile(context.Background(), mpDeleteFileReq)

	return mpDeleteFileAck.Ret

}
func (cfs *CFS) AllocateChunk(path string) (int32, *mp.ChunkInfo) {
	conn, err := grpc.Dial(metaDataAddress, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("did not connect: %v", err)
	}
	defer conn.Close()
	mc := mp.NewMetaNodeClient(conn)
	pAllocateChunkReq := &mp.AllocateChunkReq{
		FileName: path,
		VolID:    cfs.volID,
	}
	fmt.Println(pAllocateChunkReq)
	pAllocateChunkAck, _ := mc.AllocateChunk(context.Background(), pAllocateChunkReq)
	if pAllocateChunkAck.Ret != 0 {
		fmt.Print("AllocateChunk Failed!\n")
		return pAllocateChunkAck.Ret, nil
	}

	return pAllocateChunkAck.Ret, pAllocateChunkAck.ChunkInfo
}

func (cfs *CFS) GetFileChunks(path string) (int32, []*mp.ChunkInfo) {
	fmt.Println("GetFileChunks...")
	conn, err := grpc.Dial(metaDataAddress, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("did not connect: %v", err)
	}
	defer conn.Close()
	mc := mp.NewMetaNodeClient(conn)
	pGetFileChunksReq := &mp.GetFileChunksReq{
		FileName: path,
		VolID:    cfs.volID,
	}
	pGetFileChunksAck, _ := mc.GetFileChunks(context.Background(), pGetFileChunksReq)
	if pGetFileChunksAck.Ret != 0 {
		fmt.Print("GetFileChunks Failed!\n")
		return pGetFileChunksAck.Ret, nil
	}
	return pGetFileChunksAck.Ret, pGetFileChunksAck.ChunkInfos
}

type Chunk struct {
	freeSize  int32         // chunk size
	chunkInfo *mp.ChunkInfo // chunk info
	buffer    *bytes.Buffer // chunk data
}
type CFile struct {
	cfs      *CFS
	Path     string
	OpenFlag int
	fileSize int64

	// for write
	WMutex sync.Mutex
	Writer int32
	//FirstW bool
	//lastSeq  int32       // last sequence number
	wChunk   Chunk
	wChannel chan *Chunk // write channel

	// for read
	lastoffset int64
	RMutex     sync.Mutex
	//rChannel    chan []byte
	chunks  []*mp.ChunkInfo // chunkinfo
	readBuf []byte
}

func (cfile *CFile) Reads(dstFile string, offset int64, readsize int64) int64 {
	if cfile.OpenFlag != O_RDONLY {
		fmt.Println("Openflag bad parameter!\n")
		return -1
	}

	if readsize != 0 && offset != 0 {
		cfile.lastoffset = offset
		readsize = readsize
	} else if offset != 0 && readsize == 0 {
		cfile.lastoffset = offset
		readsize = cfile.fileSize - offset
	} else if offset == 0 && readsize != 0 {
		cfile.lastoffset = offset
		readsize = readsize
	} else {
		cfile.lastoffset = offset
		readsize = cfile.fileSize
	}

	//local file
	f, err := os.Create(dstFile)
	if err != nil {
		fmt.Println("Open local dstFile error!\n")
		os.Exit(1)
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	var length int64 = 0
	var ch map[int]chan *bytes.Buffer
	ch = make(map[int]chan *bytes.Buffer)

	var free_offset int64
	var free_size int64
	var begin_chunk_num int = 0
	var end_chunk_num int = 0
	cur_offset := cfile.lastoffset
	for i := range cfile.chunks {
		free_offset = cur_offset - int64(cfile.chunks[i].ChunkSize)
		if free_offset < 0 {
			begin_chunk_num = i
			break
		} else {
			cur_offset = free_offset
		}
	}

	cur_size := cfile.lastoffset + readsize
	for i := range cfile.chunks {
		free_size = cur_size - int64(cfile.chunks[i].ChunkSize)
		if free_size <= 0 {
			end_chunk_num = i
			break
		} else {
			cur_size = free_size
		}
	}

	var each_read_len int64
	freesize := readsize

	for i := range cfile.chunks[begin_chunk_num : end_chunk_num+1] {
		ch[i] = make(chan *bytes.Buffer)
		index := i + begin_chunk_num
		if cur_offset+freesize < int64(cfile.chunks[index].ChunkSize) {
			each_read_len = freesize
		} else {
			each_read_len = int64(cfile.chunks[index].ChunkSize) - cur_offset
			cur_offset = 0
		}

		freesize = freesize - each_read_len
		if i == 0 {
			fmt.Printf("== The %v-th chunk  need read size:%v (offset:%v)\n", index, each_read_len, cfile.lastoffset)
			go cfile.streamread(index, ch[i], cfile.lastoffset, each_read_len) // ch0 <- buf([]byte) ~ chn<-
		} else {
			fmt.Printf("== The %v-th chunk  need read size:%v (offset:%v)\n", index, each_read_len, cur_offset)
			go cfile.streamread(index, ch[i], cur_offset, each_read_len)
		}
	}
	var n int = 0
	for i := range cfile.chunks[begin_chunk_num : end_chunk_num+1] {

		buffer := new(bytes.Buffer)
		buffer = <-ch[i]
		buf := buffer.Next(buffer.Len())
		if n, err = w.Write(buf); err != nil {
			fmt.Printf("Write CFSfile the %v-th chunk to Localfile err:%v !\n", i, err)
			return -1
		}
		cfile.lastoffset += int64(n)
		length += int64(n)
		if err = w.Flush(); err != nil {
			fmt.Println("Flush Localfile data err!\n")
			os.Exit(1)
		}
	}
	return length
}

func (cfile *CFile) streamread(chunkidx int, ch chan *bytes.Buffer, offset int64, size int64) {
	for i, _ := range cfile.chunks[chunkidx].BlockGroup.BlockInfos {
		conn, err := grpc.Dial(utils.Inet_ntoa(cfile.chunks[chunkidx].BlockGroup.BlockInfos[i].DataNodeIP).String()+":"+strconv.Itoa(int(cfile.chunks[chunkidx].BlockGroup.BlockInfos[i].DataNodePort)), grpc.WithInsecure())
		if err != nil {
			fmt.Printf("Connect Datanode error: %v\n", err)
			continue
		}
		defer conn.Close()
		dc := dp.NewDataNodeClient(conn)
		streamreadChunkReq := &dp.StreamReadChunkReq{
			ChunkID:  cfile.chunks[chunkidx].ChunkID,
			BlockID:  cfile.chunks[chunkidx].BlockGroup.BlockInfos[i].BlockID,
			Offset:   offset,
			Readsize: size,
		}
		stream, err := dc.StreamReadChunk(context.Background(), streamreadChunkReq)
		if err != nil {
			fmt.Printf("Stream read the chunk:%v err:%v\n", cfile.chunks[chunkidx].ChunkID, err)
			close(ch)
		}
		buffer := new(bytes.Buffer)
		for {
			ack, err := stream.Recv()
			if err == io.EOF {
				//fmt.Printf("Stream read the chunkid:%v have finished!\n", cfile.chunks[chunkidx].ChunkID)
				break
			}
			if err != nil {
				fmt.Printf("Recv stream the chunkid:%v from chunksever err:%v\n", cfile.chunks[chunkidx].ChunkID, err)
				close(ch)
			}
			buffer.WriteString(ack.Databuf)
		}
		ch <- buffer
	}
}

func (cfile *CFile) Readf(data *[]byte, offset int64, readsize int64) int64 {
	if cfile.OpenFlag != O_RDONLY {
		fmt.Println("Openflag bad parameter!\n")
		return -1
	}
	if readsize != 0 && offset != 0 {
		cfile.lastoffset = offset
		if offset+readsize > cfile.fileSize {
			readsize = cfile.fileSize - offset
		} else {
			readsize = readsize
		}
	} else if offset != 0 && readsize == 0 {
		cfile.lastoffset = offset
		readsize = cfile.fileSize - offset
	} else if offset == 0 && readsize != 0 {
		cfile.lastoffset = offset
		readsize = readsize
	} else {
		cfile.lastoffset = offset
		readsize = cfile.fileSize
	}

	var length int64 = 0
	var ch chan *bytes.Buffer
	ch = make(chan *bytes.Buffer)

	var free_offset int64
	var free_size int64
	var begin_chunk_num int = 0
	var end_chunk_num int = 0
	cur_offset := offset
	for i := range cfile.chunks {
		free_offset = cur_offset - int64(cfile.chunks[i].ChunkSize)
		if free_offset < 0 {
			begin_chunk_num = i
			break
		} else {
			cur_offset = free_offset
		}
	}

	cur_size := offset + readsize

	for i := range cfile.chunks {
		free_size = cur_size - int64(cfile.chunks[i].ChunkSize)
		if free_size <= 0 {
			end_chunk_num = i
			break
		} else {
			cur_size = free_size
		}
	}

	buffer := new(bytes.Buffer)
	var each_read_len int64
	freesize := readsize

	for i := range cfile.chunks[begin_chunk_num : end_chunk_num+1] {
		index := i + begin_chunk_num
		if cur_offset+freesize < int64(cfile.chunks[index].ChunkSize) {
			each_read_len = freesize
		} else {
			each_read_len = int64(cfile.chunks[index].ChunkSize) - cur_offset
		}
		if len(cfile.readBuf) == 0 {
			go cfile.streamread(index, ch, 0, int64(cfile.chunks[index].ChunkSize))
			buffer = <-ch
			cfile.readBuf = buffer.Next(buffer.Len())
			buffer.Reset()
			//fmt.Printf("--- offset:%v  -- eachlen:%v --len:%v -- cap:%v --- \n",cur_offset,each_read_len,len(cfile.readBuf),cap(cfile.readBuf))
		}
		*data = cfile.readBuf[cur_offset : cur_offset+each_read_len]
		cur_offset += each_read_len
		if cur_offset == int64(cfile.chunks[index].ChunkSize) {
			cur_offset = 0
			cfile.readBuf = make([]byte, 0, 67108864)
		}
		freesize = freesize - each_read_len
		length += each_read_len
	}
	//var n int = 0
	//for i := range cfile.chunks[begin_chunk_num : end_chunk_num+1] {
	//	str := <-ch[i]
	//	n = len(str)
	//	*data = utils.S2B(&str)
	//	cfile.lastoffset += int64(n)
	//	length += int64(n)
	//}
	return length
}

func (cfile *CFile) Read(buf *[]byte, read_size int64) int64 {
	if cfile.OpenFlag != O_RDONLY {
		fmt.Println("Openflag bad parameter!\n")
		return -1
	}

	//cfile.RMutex.Lock()
	//defer cfile.RMutex.Unlock()

	var ret int64 = 0
	var filesize int64 = 0

	for i := range cfile.chunks {
		filesize += int64(cfile.chunks[i].ChunkSize)
	}

	if cfile.lastoffset == filesize {
		return 0
	}

	ret = cfile.Pread(buf, read_size, cfile.lastoffset)
	if ret >= 0 {
		cfile.lastoffset += ret
	}
	return ret
}

func (cfile *CFile) Pread(buf *[]byte, read_size int64, offset int64) int64 {
	if buf == nil || read_size < 0 || offset < 0 {
		fmt.Println("Pread bad parameter!\n")
		return -1
	}

	if cfile.lastoffset == -1 || cfile.lastoffset != offset {
		//cfile.sequential_ratio /= 2
		fmt.Println("Pread miss last read offset!\n")
		return -1
	}

	var free_offset int64
	var cur_chunk_num int = 0
	cur_offset := cfile.lastoffset
	for i := range cfile.chunks {
		free_offset = cur_offset - int64(cfile.chunks[i].ChunkSize)
		if free_offset < 0 {
			cur_chunk_num = i
			break
		} else {
			cur_offset = free_offset
		}
	}

	var ret_len int64
	var each_read_len int64
	var total_buf bytes.Buffer
	read_len := read_size

	for i, _ := range cfile.chunks[cur_chunk_num:] {
		cur_chunk_index := i + cur_chunk_num
		if cur_offset+read_len < int64(cfile.chunks[cur_chunk_index].ChunkSize) {
			each_read_len = read_len
		} else {
			each_read_len = int64(cfile.chunks[cur_chunk_index].ChunkSize) - cur_offset
		}
		read_len = read_len - each_read_len

		for j, _ := range cfile.chunks[cur_chunk_index].BlockGroup.BlockInfos {
			conn, err := grpc.Dial(utils.Inet_ntoa(cfile.chunks[cur_chunk_index].BlockGroup.BlockInfos[j].DataNodeIP).String()+":"+strconv.Itoa(int(cfile.chunks[cur_chunk_index].BlockGroup.BlockInfos[j].DataNodePort)), grpc.WithInsecure())
			if err != nil {
				fmt.Printf("Connect Datanode error: %v\n", err)
				continue
			}
			dc := dp.NewDataNodeClient(conn)
			readChunkReq := &dp.ReadChunkReq{
				ChunkID: cfile.chunks[cur_chunk_index].ChunkID,
				BlockID: cfile.chunks[cur_chunk_index].BlockGroup.BlockInfos[j].BlockID,
				Readlen: each_read_len,
				Offset:  cur_offset,
			}
			readChunkAck, _ := dc.ReadChunk(context.Background(), readChunkReq)
			if readChunkAck.Ret < 0 {
				fmt.Println("Read data from chunkserver return error, retry the replic chunk!\n")
				conn.Close()
			} else {
				total_buf.WriteString(readChunkAck.Databuf)
				ret_len += readChunkAck.Readsize
				//fmt.Printf("###totalchunk:%v ### curchunk:%v ### chunkid:%v ## chunksize:%v ## n_size:%v ## ret_size:%v ## offset:%v ### this time bufsize:%v ### have need read size:%v ####\n",len(cfile.chunks),cur_chunk_num,cfile.chunks[cur_chunk_index].ChunkID, cfile.chunks[cur_chunk_index].ChunkSize, each_read_len, readChunkAck.Readsize, cur_offset,read_size,read_len)
				conn.Close()
				break
			}
		}
		cur_offset += each_read_len
		if cur_offset == int64(cfile.chunks[cur_chunk_index].ChunkSize) {
			cur_offset = 0
		}
		str := total_buf.String()
		*buf = utils.S2B(&str)
		if read_len == 0 && ret_len == read_size {
			return ret_len
		}
	}

	return ret_len
}

func (cfile *CFile) Seek(offset int64, whence int32) int64 {
	if whence <= 1 {
		cfile.lastoffset = offset
	} else if whence == 2 {
		cfile.lastoffset = cfile.fileSize - offset
	} else {
		fmt.Printf("Seek file for read/write invalid whence:%v\n", whence)
		return -1
	}
	return 0
}

func (cfile *CFile) Write(buf []byte, len int32) int32 {
	cfile.WMutex.Lock()
	defer cfile.WMutex.Unlock()
	var w int32
	w = 0
	for w < len {
		if cfile.wChunk.freeSize == 0 {
			fmt.Println("need a new chunk !")
			cfile.wChunk = Chunk{buffer: new(bytes.Buffer), freeSize: chunkSize}
			var ret int32
			ret, cfile.wChunk.chunkInfo = cfile.cfs.AllocateChunk(cfile.Path)
			if ret != 0 {
				return ret
			}
		}
		if len-w < cfile.wChunk.freeSize {
			cfile.wChunk.buffer.Write(buf[w:len])
			cfile.wChunk.freeSize = cfile.wChunk.freeSize - (len - w)
			w = len
			break
		} else {
			cfile.wChunk.buffer.Write(buf[w:cfile.wChunk.freeSize])
			w = w + cfile.wChunk.freeSize
			cfile.wChunk.freeSize = 0
		}

		if cfile.wChunk.freeSize == 0 {
			cfile.WriteChunk()
		}
	}
	return w
}

func (cfile *CFile) WriteChunk() int32 {
	fmt.Println("WriteChunk in ...")
	if cfile.wChunk.chunkInfo == nil {
		return 0
	}
	cfile.wChunk.chunkInfo.ChunkSize = chunkSize - cfile.wChunk.freeSize
	wChunk := cfile.wChunk    // record cur chunk
	cfile.wChannel <- &wChunk // push to channel
	cfile.wChunk.freeSize = 0 // disable cur chunk
	Wg.Add(1)
	return 0
}
func (cfile *CFile) flushChannel() {
	for {
		v := <-cfile.wChannel
		fmt.Println("channel has a num !!!")
		for i := range v.chunkInfo.BlockGroup.BlockInfos {
			addr := utils.Inet_ntoa(v.chunkInfo.BlockGroup.BlockInfos[i].DataNodeIP).String() + ":" + strconv.Itoa(int(v.chunkInfo.BlockGroup.BlockInfos[i].DataNodePort))
			//fmt.Println("addr:")
			//fmt.Println(addr)
			conn, err := grpc.Dial(addr, grpc.WithInsecure())
			if err != nil {
				fmt.Printf("did not connect: %v", err)
			}
			defer conn.Close()
			dc := dp.NewDataNodeClient(conn)

			dataBuf := v.buffer.String()
			dataLen := len(dataBuf)
			//fmt.Println(dataLen)
			blockID := v.chunkInfo.BlockGroup.BlockInfos[i].BlockID
			chunkID := v.chunkInfo.ChunkID

			stream, err1 := dc.WriteChunkStream(context.Background())
			if err1 != nil {
				fmt.Printf("stream.Send error !")
			}
			var cur int = 0
			for cur < dataLen {
				if dataLen-cur < 1024*1024 {
					pWriteChunkReq := &dp.WriteChunkReq{
						ChunkID: chunkID,
						BlockID: blockID,
						Databuf: dataBuf[cur:dataLen],
					}
					if err := stream.Send(pWriteChunkReq); err != nil {
						fmt.Printf("stream.Send error !")
					}
					cur = dataLen
				} else {
					pWriteChunkReq := &dp.WriteChunkReq{
						ChunkID: chunkID,
						BlockID: blockID,
						Databuf: dataBuf[cur : cur+1024*1024],
					}
					if err := stream.Send(pWriteChunkReq); err != nil {
						fmt.Printf("stream.Send error !")
					}
					cur = cur + 1024*1024
				}

			}
			stream.CloseAndRecv()

		}

		fmt.Println("update chunkinfo to metanode...")
		// update chunkinfo to metanode
		conn, err := grpc.Dial(metaDataAddress, grpc.WithInsecure())
		if err != nil {
			fmt.Printf("did not connect: %v", err)
		}

		defer conn.Close()
		mc := mp.NewMetaNodeClient(conn)
		pSyncChunkReq := &mp.SyncChunkReq{
			FileName:  cfile.Path,
			VolID:     cfile.cfs.volID,
			ChunkInfo: v.chunkInfo,
		}
		pSyncChunkAck, _ := mc.SyncChunk(context.Background(), pSyncChunkReq)
		if pSyncChunkAck.Ret != 0 {
			fmt.Print("SyncChunk Failed!\n")
			return
		}
		Wg.Add(-1)
	}
}

func (cfile *CFile) Flush() int32 {
	fmt.Println("Flush  ... ")

	if cfile.OpenFlag != O_RDONLY {
		cfile.WriteChunk()
	}
	return 0
}

func (cfile *CFile) Sync() int32 {
	return 0
}

func (cfile *CFile) Close() int32 {
	fmt.Println("Close  ... ")

	if cfile.OpenFlag != O_RDONLY {
		cfile.WriteChunk()
	}
	return 0
}

func ProcessLocalBuffer(buffer []byte, cfile *CFile) {
	//fmt.Println("ProcessLocalBuffer...")
	cfile.Write(buffer, int32(len(buffer)))
}

func ReadLocalAndWriteCFS(filePth string, bufSize int, hookfn func([]byte, *CFile), cfile *CFile) error {
	f, err := os.Open(filePth)
	if err != nil {
		return err
	}
	defer f.Close()
	buf := make([]byte, bufSize)
	bfRd := bufio.NewReader(f)
	for {
		n, err := bfRd.Read(buf)
		hookfn(buf[:n], cfile)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
	return nil
}