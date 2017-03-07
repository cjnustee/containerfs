package main

import (
	"bufio"
	"fmt"
	fs "ipd.org/containerfs/fs"
	"ipd.org/containerfs/utils"
	"os"
	"strconv"
)

func main() {

	switch os.Args[1] {

	case "createvol":
		argNum := len(os.Args)
		if argNum != 4 {
			fmt.Println("createvol [volname] [space GB]")
			os.Exit(1)
		}
		fs.CreateVol(os.Args[2], os.Args[3])

	case "getvolinfo":
		argNum := len(os.Args)
		if argNum != 3 {
			fmt.Println("getvolinfo [volUUID]")
			os.Exit(1)
		}
		ret, vi := fs.GetVolInfo(os.Args[2])
		if ret == 0 {
			fmt.Println(vi)
		} else {
			fmt.Printf("get volume info failed , ret :%d", ret)
		}

	case "createdir":
		argNum := len(os.Args)
		if argNum != 4 {
			fmt.Println("createdir [volUUID] [dirname]")
			os.Exit(1)
		}
		cfs := fs.OpenFileSystem(os.Args[2])
		ret := cfs.CreateDir(os.Args[3])
		if ret == -1 {
			fmt.Print("create dir failed\n")
			return
		}
		if ret == 1 {
			fmt.Print("not allowed\n")
			return
		}
		if ret == 2 {
			fmt.Print("no parent path\n")
			return
		}
		if ret == 17 {
			fmt.Print("already exist\n")
			return
		}

	case "stat":
		argNum := len(os.Args)
		if argNum != 4 {
			fmt.Println("stat [volUUID] [dir/filename]")
			os.Exit(1)
		}
		cfs := fs.OpenFileSystem(os.Args[2])
		ret, inode := cfs.Stat(os.Args[3])
		if ret == 0 {
			fmt.Println(inode)
		} else if ret == 2 {
			fmt.Println("not found")
		} else {
			fmt.Println("stat failed")
		}

	case "ls":
		argNum := len(os.Args)
		if argNum != 4 {
			fmt.Println("ls [volUUID] [dir/filename]")
			os.Exit(1)
		}
		cfs := fs.OpenFileSystem(os.Args[2])
		ret, inodes := cfs.List(os.Args[3])

		if ret == 0 {
			for _, value := range inodes {
				fmt.Println(value.Name)
			}
		} else if ret == 2 {
			fmt.Println("not found")
		} else {
			fmt.Println("ls failed")
		}

	case "ll":
		argNum := len(os.Args)
		if argNum != 4 {
			fmt.Println("ls [volUUID] [dir/filename]")
			os.Exit(1)
		}
		cfs := fs.OpenFileSystem(os.Args[2])
		ret, inodes := cfs.List(os.Args[3])

		if ret == 0 {
			for _, value := range inodes {
				fmt.Println(value)
			}
		} else if ret == 2 {
			fmt.Println("not found")
		} else {
			fmt.Println("ls failed")
		}

	case "deletedir":
		argNum := len(os.Args)
		if argNum != 4 {
			fmt.Println("deletedir [volUUID] [dirname]")
			os.Exit(1)
		}
		cfs := fs.OpenFileSystem(os.Args[2])
		ret := cfs.DeleteDir(os.Args[3])
		if ret != 0 {
			if ret == 2 {
				fmt.Println("not allowed")
			} else {
				fmt.Println("delete dir failed")
			}
		}

	case "mv":
		argNum := len(os.Args)
		if argNum != 5 {
			fmt.Println("mv [volUUID] [dirname1] [dirname2]")
			os.Exit(1)
		}
		cfs := fs.OpenFileSystem(os.Args[2])
		ret := cfs.Rename(os.Args[3], os.Args[4])
		if ret == 2 {
			fmt.Println("not existed")
		}
		if ret == 1 {
			fmt.Println("not allowed")
		}
	case "touch":
		argNum := len(os.Args)
		if argNum != 4 {
			fmt.Println("touch [volUUID] [filename]")
			os.Exit(1)
		}
		cfs := fs.OpenFileSystem(os.Args[2])
		ret, _ := cfs.OpenFile(os.Args[3], fs.O_WRONLY)
		if ret != 0 {
			fmt.Println("touch failed")
		}

	case "deletefile":
		argNum := len(os.Args)
		if argNum != 4 {
			fmt.Println("deletedir [volUUID] [filename]")
			os.Exit(1)
		}
		cfs := fs.OpenFileSystem(os.Args[2])
		ret := cfs.DeleteFile(os.Args[3])
		if ret != 0 {
			if ret == 2 {
				fmt.Println("not found")
			} else {
				fmt.Println("delete file failed")
			}
		}
	case "allocatechunk":
		argNum := len(os.Args)
		if argNum != 4 {
			fmt.Println("allocatechunk [volUUID] [filename]")
			os.Exit(1)
		}
		cfs := fs.OpenFileSystem(os.Args[2])
		ret, ack := cfs.AllocateChunk(os.Args[3])

		if ret != 0 {
			fmt.Println("allocatechunk failed")
		} else {
			fmt.Println(ack)
		}

	case "get1":
		argNum := len(os.Args)
		if argNum != 5 {
			fmt.Println("get [voluuid] [cfsfilename] [dstfilename]")
			os.Exit(1)
		}
		cfs := fs.OpenFileSystem(os.Args[2])
		get(cfs, os.Args[3], os.Args[4])

	case "put":
		argNum := len(os.Args)
		if argNum != 5 {
			fmt.Println("put [volUUID] [localfilename] [cfsfilename]")
			os.Exit(1)
		}
		cfs := fs.OpenFileSystem(os.Args[2])
		put(cfs, os.Args[3], os.Args[4])

	case "get2":
		argNum := len(os.Args)
		if argNum != 7 {
			fmt.Println("get [voluuid] [cfsfilename] [dstfilename] [offset] [readsize]")
			os.Exit(1)
		}
		offset, _ := strconv.ParseInt(os.Args[5], 10, 64)
		size, _ := strconv.ParseInt(os.Args[6], 10, 64)

		cfs := fs.OpenFileSystem(os.Args[2])
		getstream(cfs, os.Args[3], os.Args[4], offset, size)
	}

	fs.Wg.Wait()
}

func getstream(cfs *fs.CFS, cfsFile string, dstFile string, offset int64, readsize int64) {
	ret, _ := cfs.Stat(cfsFile)
	if ret != 0 {
		fmt.Print("Get Bad FilePath from CFS!\n")
		os.Exit(1)
	}

	ret, cfile := cfs.OpenFile(cfsFile, fs.O_RDONLY)
	defer cfile.Close()

	var length int64 = 0
	length = cfile.Reads(dstFile, offset, readsize)
	fmt.Printf("Read %v bytes from %s have finised... !\n", length, cfsFile)
}

func get(cfs *fs.CFS, cfsFile string, dstFile string) {
	ret, _ := cfs.Stat(cfsFile)
	if ret != 0 {
		fmt.Print("Get Bad FilePath from CFS!\n")
		os.Exit(1)
	}

	ret, cfile := cfs.OpenFile(cfsFile, fs.O_RDONLY)
	defer cfile.Close()

	f, err := os.Create(dstFile)
	if err != nil {
		fmt.Println("Open local dstFile error!\n")
		os.Exit(1)
	}
	defer f.Close()
	w := bufio.NewWriter(f)

	buf := make([]byte, 1024*1024)
	var bytes int64 = 0
	var length int64 = 0
	for {
		length = cfile.Read(&buf, int64(len(buf)))
		if length <= 0 {
			if length < 0 {
				fmt.Println("Read from CFSFile fail!\n")
				os.Exit(1)
			}
			break
		}
		bytes += length
		if n, err := w.Write(buf); err != nil {
			fmt.Printf("Get from CFSfile to Localfile err:%v !\n", err)
			os.Exit(1)
		} else if int64(n) != length {
			fmt.Printf("Get from CFSfile to write Localfile incorrect, retsize:%v, writesize:%v !!!\n", length, n)
			os.Exit(1)
		}
	}
	if err = w.Flush(); err != nil {
		fmt.Println("Flush Localfile data err!\n")
		os.Exit(1)
	}
	fmt.Printf("Read %v bytes from %s have finised...\n", bytes, cfsFile)
}

func put(cfs *fs.CFS, localFile string, cfsFile string) int32 {
	if ok, _ := utils.LocalPathExists(localFile); !ok {
		fmt.Println("local file not exist!")
		os.Exit(1)
	}
	ret, cfile := cfs.OpenFile(cfsFile, fs.O_WRONLY)
	if ret != 0 {
		return ret
	}
	fs.ReadLocalAndWriteCFS(localFile, 1024*10, fs.ProcessLocalBuffer, cfile)
	cfile.Close()

	return 0
}