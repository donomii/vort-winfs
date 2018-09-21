package main

import (
"os"
"encoding/json"
	"io/ioutil"
	"runtime/debug"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	//"os"
	"runtime"
	"strconv"
	"strings"
	"time"
	"sync"

	"github.com/donomii/hashare"

	"github.com/keybase/dokan-go"
	"github.com/keybase/kbfs/dokan/winacl"
	//"github.com/keybase/kbfs/ioutil"
	//"golang.org/x/sys/windows"
)


type VortFile struct {
	hashare.VortFile
		WriteMutex 	sync.Mutex
	}

	var GlobalWriteMutex sync.Mutex
type VortFS struct{
	NextFileHandle	uint64
	Config		*hashare.Config
    FileMeta    hashare.FileCache

}

var fs VortFS

func main() {
	fmt.Println("vort-winfs started")
	GlobalWriteMutex = sync.Mutex{}
	go func() {
		for {
			//var m runtime.MemStats
			//runtime.ReadMemStats(&m)
			//log.Printf("\nAlloc = %v\nTotalAlloc = %v\nSys = %v\nNumGC = %v\n\n", m.Alloc/1024, m.TotalAlloc/1024, m.Sys/1024, m.NumGC)
			time.Sleep(1 * time.Second)
		}
	}()
	/*
	drive := os.Args[1]
	repository = os.Args[2]
	*/
	
	var userconfig Config
	raw, err := ioutil.ReadFile("vort.config")
	if err != nil {
		panic(err)
	}

	json.Unmarshal(raw, &userconfig)
	
	drive := userconfig.Mount
	repository := userconfig.Repository
	fmt.Println("Attempting to mount", repository, "on", drive)
	
	fs = VortFS{FileMeta: hashare.FileCache{}}
	
	fs.FileMeta.Init()
	
	Conf()
	mnt, err := dokan.Mount(&dokan.Config{FileSystem: &fs, Path: drive, MountFlags: dokan.Network | dokan.Removable})
	//mnt, err := dokan.Mount(&dokan.Config{FileSystem: &fs, Path: drive, MountFlags: dokan.Network})
	//mnt, err := dokan.Mount(&dokan.Config{FileSystem: &fs, Path: drive})
	if err != nil {
		log.Fatal("Mount failed:", err)
	}
	log.Println("Mount successful, ready to serve")
	for {
		time.Sleep(1 * time.Millisecond)
	}
	defer mnt.Close()
}

var _ dokan.FileSystem = VortFS{}

func dbg(s string) {
	log.Println(s)
}

func dbgcall(s string) {
	log.Println("(fusecall)" + s)
}



func (t VortFile) GetFileSecurity(ctx context.Context, fi *dokan.FileInfo, si winacl.SecurityInformation, sd *winacl.SecurityDescriptor) error {
	dbg("VortFS.GetFileSecurity")
	return nil
}
func (t VortFile) SetFileSecurity(ctx context.Context, fi *dokan.FileInfo, si winacl.SecurityInformation, sd *winacl.SecurityDescriptor) error {
	dbg("VortFS.SetFileSecurity")
	return nil
}
func (t VortFile) Cleanup(ctx context.Context, fi *dokan.FileInfo) {
	dbg("VortFS.Cleanup:" + fi.Path())
	fs.Flush(fi.Path(), 0)
}

func (t VortFile) CloseFile(ctx context.Context, fi *dokan.FileInfo) {
	defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovered in CloseFile", r)
			dbg(fmt.Sprintf("Recovered in CloseFile: %v", r))
			log.Printf("%s: %s", r, debug.Stack()) 
        }
    }()
	path := toUnix(fi.Path())
	dbg("VortFS.CloseFile: " + path)
	if fi.IsDeleteOnClose() {
		path := strings.Replace(path, "\\", "/", -1)
		dbgcall("CanDeleteFile " + path )
		conf := Conf()
		var ok bool
		hashare.WithTransaction(Conf(), "Delete file during close: " + path, func(tr hashare.Transaction) hashare.Transaction {
			tr, ok = hashare.DeleteFile(conf.Store, path, conf, tr)
			if !ok {
				dbg("Could not delete file:"+path)
				panic("Could not delete file:"+path)
				//If the file doesn't exist, we don't fail, we just create it
				//return -fuse.EIO
			}
			return tr
				})
	} else {
		fs.Flush(path, 0)
	}
	refcount := fs.FileMeta.Close(path)
	dbg(fmt.Sprintf("Closed %v with refcount now %v", path, refcount))
}

func (t VortFS) WithContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return ctx, nil
}

func (t VortFS) GetVolumeInformation(ctx context.Context) (dokan.VolumeInformation, error) {
	//dbg("VortFS.GetVolumeInformation")
	return dokan.VolumeInformation{
		VolumeName:             "VORT",
		MaximumComponentLength: 0xFF, // This can be changed.
		FileSystemFlags: dokan.FileCasePreservedNames | dokan.FileCaseSensitiveSearch |
			dokan.FileUnicodeOnDisk, //| dokan.FileSequentalWriteOnce,
		FileSystemName: "VORT",
	}, nil
}

var dummyFreeSpace uint64 = 512 * 1024 * 1024 * 1024

func freeSpace() dokan.FreeSpace {
	return dokan.FreeSpace{
		TotalNumberOfBytes:     dummyFreeSpace * 4,
		TotalNumberOfFreeBytes: dummyFreeSpace * 3,
		FreeBytesAvailable:     dummyFreeSpace * 3,
	}

}
func (t VortFS) GetDiskFreeSpace(ctx context.Context) (dokan.FreeSpace, error) {
	dbg("VortFS.GetDiskFreeSpace")
	return freeSpace(), nil
}

func (t VortFS) ErrorPrint(err error) {
	dbg(fmt.Sprintf("%v", err))
}


func (self *VortFS) MakeFile(path string, size int) {
		dbg("MakeFile "+path)
        m := hashare.VortFile{}
        m.Data = make([]byte, size)
        m.Loaded = true
		m.Dirty = true
		self.FileMeta.SetVal(path, &m)
		dbg("MakeFile complete: "+path)
}


func int2perms(bperms uint32) string {
	bitstring := strconv.FormatInt(int64(bperms), 2)
	bits := strings.Split(bitstring, "")
	for i, j := 0, len(bits)-1; i < j; i, j = i+1, j-1 {
		bits[i], bits[j] = bits[j], bits[i]
	}
	pattern := "xwrxwrxwrtgu**********"
	var out string
	for i, v := range bits {
		flag := string(pattern[i])
		if v == "1" {
		if flag != "*" { //Unused bit
			out =   string(flag) + out 
			}
		} else {
			if flag != "*" { //Unused bit
				out = "-" + out
			}
		}
	}
	for _=1;len(out)<12; out = "-"+out {}
	return out
}


func (Self *VortFS) Chmod(path string, mode uint32) (errc int) {
	defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovered in Chmod", r)
			dbg(fmt.Sprintf("Recovered in Chmod: %v", r))
			log.Printf("%s: %s", r, debug.Stack()) 
			errc = -1
        }
    }()
	conf := Conf()
	dbgcall("Chmod " + path + " to " + int2perms(mode))
	tr := hashare.BeginTransaction(conf)
	meta, ok := hashare.GetMeta(path, conf, tr)
	
	if !ok {
		dbg("(Chmod) File not found:"+ path)
		return -1
	}
	dbg(fmt.Sprintf("(Chmod) Got meta for: %v, %v", path, meta))
	modeStr := int2perms(mode)
	meta.Permissions = modeStr
	tr, ok = hashare.SetMeta(path, *meta, conf, tr)
	if !ok {
		dbg("(Chmod) Set file attributes failed:"+ path)
		return -1
	}
	ok = hashare.CommitTransaction(tr, "Chmod " + path, conf)
	if !ok {
		dbg("(Chmod) Commit transaction failed:"+ path)
		return -1
	}
	return 0	
}

func (self *VortFS) Release(path string, fh uint64) (errc int) {
	defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovered in Release", r)
			log.Printf("%s: %s", r, debug.Stack()) 
			errc = -1
        }
    }()
	dbgcall("Release " + path)
	self.Flush(path, fh)
	dbg("Released " + path)
	return 0
}

func (self *VortFS) Flush(path string, fh uint64) (errc int) {
	GlobalWriteMutex.Lock()
	defer GlobalWriteMutex.Unlock()
	defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovered in Flush", r)
			log.Printf("%s: %s", r, debug.Stack()) 
			errc = -1
        }
    }()
	unixPath := strings.Replace(path, "\\", "/", -1)
	path = unixPath
	dbgcall("Flush " + path)
	conf := Conf()
	returnVal := 0
	meta, ok := self.FileMeta.GetVal(path) 
    if ok && meta.Dirty {
		hashare.WithTransaction(Conf(), "Flush file" + unixPath, func(tr hashare.Transaction) hashare.Transaction {
			tr, ok = hashare.DeleteFile(conf.Store, path, conf, tr)
			/*
			Ignore delete failures in flush, the file might not exist yet
			if !ok {
				dbg("Could not delete file in flush:"+path)
				returnVal = -1
				panic("Could not delete file in flush:"+path)
				//If the file doesn't exist, we don't fail, we just create it
				//return -fuse.EIO
			}
			
			dbg("(Flush) Delete complete:"+path)
			*/
			tr, ok = hashare.PutBytes(conf.Store, meta.Data, path, conf, true, tr)
			if !ok {
				dbg("Could not put file in flush:"+path)
				returnVal = -1
				panic("Could not put file in flush:"+path)
			}
			dbg("(Flush) PutBytes complete:" + path)
			meta.Dirty = false
			self.FileMeta.SetVal(path, meta)
			return tr
		})
		
    } else {
		dbg("File hasn't been opened or hasn't been modified, skipping flush")
	}
	dbg("Flushed " + path)
	return returnVal
}

func checkfor(str string, checkcode, code uint32, codes []string ) []string {
	if code & checkcode > 0 {
		codes = append(codes, str)
	}
	return codes
}

func showCreateOptions ( code uint32 ) []string {
	var out []string
	
out = checkfor("FILE_WRITE_THROUGH",   0x00000002, code, out)
out = checkfor("FILE_SEQUENTIAL_ONLY", 0x00000004,code ,out)
out = checkfor("FILE_NO_INTERMEDIATE_BUFFERING", 0x00000008,code ,out)
out = checkfor("FILE_SYNCHRONOUS_IO_ALERT", 0x00000010,code ,out)
out = checkfor("FILE_SYNCHRONOUS_IO_NONALERT", 0x00000020,code ,out)
out = checkfor("FILE_NON_DIRECTORY_FILE", 0x00000040,code ,out)
out = checkfor("FILE_CREATE_TREE_CONNECTION", 0x00000080,code ,out)
out = checkfor("FILE_COMPLETE_IF_OPLOCKED", 0x00000100,code ,out)
out = checkfor("FILE_NO_EA_KNOWLEDGE", 0x00000200,code ,out)
out = checkfor("FILE_OPEN_FOR_RECOVERY", 0x00000400,code ,out)
out = checkfor("FILE_RANDOM_ACCESS", 0x00000800,code ,out)
out = checkfor("FILE_DELETE_ON_CLOSE", 0x00001000,code ,out)
out = checkfor("FILE_OPEN_BY_FILE_ID", 0x00002000,code ,out)
out = checkfor("FILE_OPEN_FOR_BACKUP_INTENT", 0x00004000,code ,out)
out = checkfor("FILE_NO_COMPRESSION", 0x00008000,code ,out)
out = checkfor("FILE_OPEN_REQUIRING_OPLOCK", 0x00010000,code ,out)
out = checkfor("FILE_DISALLOW_EXCLUSIVE", 0x00020000,code ,out)
out = checkfor("FILE_SESSION_AWARE", 0x00040000,code ,out)
out = checkfor("FILE_RESERVE_OPFILTER", 0x00100000,code ,out)
out = checkfor("FILE_OPEN_REPARSE_POINT", 0x00200000,code ,out)
out = checkfor("FILE_OPEN_NO_RECALL", 0x00400000,code ,out)
out = checkfor("FILE_OPEN_FOR_FREE_SPACE_QUERY", 0x00800000,code ,out)
out = checkfor("FILE_COPY_STRUCTURED_STORAGE", 0x00000041,code ,out)
out = checkfor("FILE_STRUCTURED_STORAGE", 0x00000441, code, out)
return out
}

func showCreateDisposition ( code int ) string {
switch dokan.CreateDisposition(code) {
			//OPEN_ALWAYS
			case dokan.CreateDisposition(dokan.FileOpenIf):  
				return "OPEN_ALWAYS (FileOpenIf)"
			//CREATE_ALWAYS
			case dokan.CreateDisposition(dokan.FileSupersede):  
				return "CREATE_ALWAYS (FileSupersede)"
			//TRUNCATE_EXISTING
			case dokan.CreateDisposition(dokan.FileOverwrite):
				return "TRUNCATE_EXISTING (FileOverwrite)"
			//CREATE_ALWAYS
			case dokan.CreateDisposition(dokan.FileOverwriteIf):
				return "CREATE_ALWAYS (FileOverwriteIf)"
			//CREATE_NEW
			case dokan.CreateDisposition(dokan.FileCreate):
				return "CREATE_NEW (FileCreate)"
			case dokan.CreateDisposition(dokan.FileOpen):
				return "OPEN_EXISTING (FileOpen)"
			default:
				dbg(fmt.Sprintf("Unhandled disposition: %v\n", code))
			}
			return(fmt.Sprintf("Unhandled disposition: %v\n", code))
}

func toUnix(str string) string {
	return strings.Replace(str, "\\", "/", -1)
}

func (self VortFS) CreateFile(ctx context.Context, fi *dokan.FileInfo, cd *dokan.CreateData) (file dokan.File, isDirectory bool, err error) {

	unixPath := toUnix(fi.Path())
	path := unixPath
	defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovered in Open", r)
			log.Printf("%s: %s", r, debug.Stack()) 
			file = VortFile{}
			isDirectory = false
			err = errors.New("panic recovery")
        }
    }()
	
	dbgcall(fmt.Sprintf("Createfile %v, %v, %v", strings.Join(showCreateOptions(cd.CreateOptions), ","), showCreateDisposition(int(cd.CreateDisposition)), path ))
	fs.FileMeta.Open(unixPath)
	conf := Conf()
	switch path {
	case `/`, `\`, ``:
		if cd.CreateOptions&dokan.FileNonDirectoryFile != 0 {
			dbg("Createfile: Returning dokan.ErrFileIsADirectory")
			return nil, true, dokan.ErrFileIsADirectory
		}
		dbg("Createfile: Returning opened directory:"+unixPath)
		return VortFile{}, true, nil
	default:
		if (cd.CreateOptions & dokan.FileDirectoryFile) > 0 {
			dbg("File type is directory")
			hashare.WithTransaction(Conf(), "make  directory" + unixPath, func(tr hashare.Transaction) hashare.Transaction {
				ret, _ := hashare.MkDir(Conf().Store, unixPath, Conf(), tr)
				return ret
			})
		} else {
			dbg("Not a directory")
			switch dokan.CreateDisposition(cd.CreateDisposition) {
			//OPEN_ALWAYS
			case dokan.CreateDisposition(dokan.FileOpenIf):  
				dbg("Open file, or create then open: " + path)
				dbg("Createfile: Conditional open:"+unixPath)
				_, ok := hashare.GetCurrentMeta(unixPath, Conf())
				if !ok {
				self.MakeFile(path, 0)
				err := self.Flush(path, 0)
				if err != 0 {
					panic(fmt.Sprintf("Could not Flush %v because %v", path, err ))
				}
				err = self.Release(path, 0)
				if err != 0 {
					panic("Could not Release " + path)
				}
				err = self.Chmod(path, 0777) //FIXME
				if err != 0 {
					panic("Could not chmod " + path)
				}
				dbg("Created " + path)
			}
			//CREATE_ALWAYS
			case dokan.CreateDisposition(dokan.FileSupersede): 
				dbg("Delete then open: " + path)
				//If the file exists, replace it, otherwise create it
				dbg("Createfile: Supersede:"+unixPath)
				hashare.WithTransaction(Conf(), "Delete file: " + path, func(tr hashare.Transaction) hashare.Transaction {
				tr, _ = hashare.DeleteFile(conf.Store, path, conf, tr)
				self.MakeFile(path, 0)
				return tr
			})
			//TRUNCATE_EXISTING
			case dokan.CreateDisposition(dokan.FileOverwrite):
				dbg("Createfile: Overwrite(truncate to 0):"+unixPath)
				_, ok := hashare.GetCurrentMeta(unixPath, Conf())
				if !ok {
					dbg("Createfile: file exists, will not overwrite: "+unixPath)
					return VortFile{}, false, errors.New("Createfile: File exists")
				}
			//CREATE_ALWAYS
			case dokan.CreateDisposition(dokan.FileOverwriteIf):
				dbg("Createfile: Create or open at 0:"+unixPath)
				_, ok := hashare.GetCurrentMeta(unixPath, Conf())
				hashare.WithTransaction(Conf(), "Delete file: " + path, func(tr hashare.Transaction) hashare.Transaction {
				tr, ok = hashare.DeleteFile(conf.Store, path, conf, tr)
				self.MakeFile(path, 0)
				return tr
			})
			//CREATE_NEW
			case dokan.CreateDisposition(dokan.FileCreate):
				dbg("Createfile: Creating empty file:"+unixPath)
				_, ok := hashare.GetCurrentMeta(unixPath, Conf())
				if ok {
					return VortFile{}, false, errors.New("File already exists")
				}
				if !ok {
				self.MakeFile(path, 0)
				err := self.Flush(path, 0)
				if err != 0 {
					panic(fmt.Sprintf("Could not Flush %v because %v", path, err ))
				}
				err = self.Release(path, 0)
				if err != 0 {
					panic("Could not Release " + path)
				}
				err = self.Chmod(path, 0777) //FIXME
				if err != 0 {
					panic("Could not chmod " + path)
				}
				dbg("Created " + path)
			}
			//OPEN_EXISTING
			case dokan.CreateDisposition(dokan.FileOpen):
				dbg("Createfile: Opening:"+unixPath)
				_, ok := hashare.GetCurrentMeta(unixPath, Conf())
				if !ok {
					dbg("Createfile: file does not exist: "+unixPath)
					return VortFile{}, false, errors.New("Createfile: File does not exist")
				}
			default:
				dbg(fmt.Sprintf("Unhandled disposition: %v\n", cd.CreateOptions))
			}
		}
		f, ok := hashare.GetCurrentMeta(unixPath, Conf())
		if !ok {
			dbg("Weird error here")
			return VortFile{}, false, errors.New("Createfile: Weird error here")
			panic("Weird error here")
		}
		log.Printf("Got metadata: %+v", f)
		if string(f.Type) == "dir" {
			log.Println("Returning dir for:", unixPath)
			if cd.CreateOptions&dokan.FileNonDirectoryFile != 0 {
			dbg("Createfile: Returning directory: "+unixPath)
				return nil, true, dokan.ErrFileIsADirectory
			}
			dbg("Returning file: "+ unixPath)
			VortFile{}.CheckLoaded(unixPath)
			return VortFile{}, false, nil
		}
		log.Println("Returning file for:", unixPath)
		dbg("Returning file for: "+ unixPath)
		VortFile{}.CheckLoaded(unixPath)
		dbg("Opened " + path + ", FH is " + fmt.Sprintf("%v",self.NextFileHandle))
		return VortFile{}, false, nil
		}
	VortFile{}.CheckLoaded(unixPath)
	
	return
}

func (t VortFile) CanDeleteFile(ctx context.Context, fi *dokan.FileInfo) error {
	path := toUnix(fi.Path())
	dbgcall("CanDeleteFile " + path )
	return nil
}

func (t VortFile) CanDeleteDirectory(ctx context.Context, fi *dokan.FileInfo) error {
	path := toUnix(fi.Path())
	dbgcall("CanDeleteDirectory " + path )
	return nil
}
func (self VortFile) SetEndOfFile(ctx context.Context, fi *dokan.FileInfo, length int64) (err error) {
	defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovered in SetEndOfFile", r)
			log.Printf("Recovered in SetEndOfFile\n%s: %s", r, debug.Stack()) 
			err = errors.New("Recovered int setendoffile")
        }
    }()
	
	path := toUnix(fi.Path())
	dbgcall("(SetEndOfFile) Truncate " + path + " to " + fmt.Sprintf("%v", length))
	self.CheckLoaded(path)
	meta, ok := fs.FileMeta.GetVal(path)
	if !ok {
		panic("Couldn't load: " + path)
	}
	data := meta.Data
	switch {
	case int(length) < len(data):
		data = data[:int(length)]
	case int(length) > len(data):
		data = append(data, make([]byte, int(length)-len(data))...)
	}
	
	
	meta.Data = data
	fs.FileMeta.SetVal(path, meta)
	dbg(fmt.Sprintf("New file length in cache is %v", len(data)))
	
	dbg("(SetEndOfFile) Finished:" + path)
	
	return nil
}
func (t VortFile) SetAllocationSize(ctx context.Context, fi *dokan.FileInfo, length int64) (err error) {
	defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovered in SetAllocationsize", r)
			log.Printf("%s: %s", r, debug.Stack())
			err = errors.New("Recovered in SetAllocatiionSize")
        }
    }()
	path := toUnix(fi.Path())
	dbg("VortFile.SetAllocationSize Start" + path)
	data, ok := hashare.GetFile(Conf().Store, path, 0, -1, Conf())
	if !ok {
		data = []byte{}
	}
	switch {
	case int(length) < len(data):
		data = data[:int(length)]
	}
	transaction := hashare.BeginTransaction(Conf())
	transaction, ok = hashare.DeleteFile(Conf().Store, path, Conf(),  transaction)
	transaction, ok = hashare.PutBytes(Conf().Store, data, path, Conf(), true, transaction)
	hashare.CommitTransaction(transaction, "Set allocation size", Conf())
	if !ok {
		log.Println("SetAllocatiionSize: Couldn't save:", path)
		return errors.New("Couldn't write")
	}
	dbg("VortFile.SetAllocationSize Finish" + path)
	return nil
}

func (t VortFS) MoveFile(ctx context.Context, src dokan.File, sourceFI *dokan.FileInfo, targetPath string, replaceExisting bool) error {
	sourcePath := strings.Replace(sourceFI.Path(), "\\", "/", -1)
	dbgcall("VortFS.MoveFile " + sourcePath	)
	panic("Not implemented")
	return nil
}

func (t VortFile) ReadFile(ctx context.Context, fi *dokan.FileInfo, bs []byte, offset int64) (n int, err error) {
	defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovered in ReadFile", r)
			log.Printf("%s: %s", r, debug.Stack())
			n = 0
			err = errors.New("Recovered in Readfile")
        }
    }()
	path := toUnix(fi.Path())
	err = nil
	dbgcall(fmt.Sprintf("ReadFile %v. Want %v bytes from offset %v", path, len(bs), offset  ))
	//conf := Conf()
	start := offset
	finish := offset + int64(len(bs))
	//dbg(fmt.Sprintf("ReadFile: %v - %v, %v", offset, finish, path))
	log.Printf("ReadFile: %v - %v, %v", offset, finish, path)
	t.CheckLoaded(path)
	file, ok := fs.FileMeta.GetVal(path)
	data := file.Data
	if !ok {
		log.Println("File not found:", path)
		return 0, dokan.ErrObjectNameNotFound //FIXME different error types
	}
	if start > int64(len(data)) {
		err = io.EOF
		dbg(fmt.Sprintf("Caught read at end of file, returning 0"))
		return 0, err
	}
	if finish > int64(len(data)) {
		err = io.EOF
		dbg(fmt.Sprintf("Caught read past end of file, reducing end of read from %v to %v", finish, len(data)))
		finish = int64(len(data))
	}
	if finish == int64(len(data)) {
		err = io.EOF
		dbg(fmt.Sprintf("Caught read at end of file, reducing end of read from %v to %v", finish, len(data)))
	}
	//dbg("Got data " + string(data))
	//rd := bytes.NewReader(data)
	dbg(fmt.Sprintf("ReadFile Complete: %v - %v, %v", offset, finish, path))
	count := copy(bs, data[offset:finish])
	n = count
	dbg(fmt.Sprintf("ReadFile: copied %v bytes,  %v - %v, with error %v", count, offset, finish, err))
	return count, err
}


func (self *VortFS) LoadFile(path string) {
		dbg("LoadFile " + path)
        data, ok := hashare.GetFile(Conf().Store, path, 0, -1, Conf())
		if !ok {
			fmt.Println("Loadfile failed! " + path)
		}
        m := hashare.VortFile{}
        m.Data = data
        m.Loaded = true
		self.FileMeta.SetVal(path, &m)
		dbg("LoadFile complete: "+path)
}

func (self VortFile) CheckLoaded(path string) {
	meta, ok := fs.FileMeta.GetVal(path)
	dbg(fmt.Sprintf("Checkloaded: %v (%v)", path, ok))
	if !(ok && meta.Loaded) {
        fs.LoadFile(path)
	}
}


func (self VortFile) WriteLock() {
	dbg("Waiting on lock")
	GlobalWriteMutex.Lock()
}

func (self VortFile) WriteUnLock() {
	dbg("Releasing lock")
	GlobalWriteMutex.Unlock()
}


func (self VortFile) WriteFile(ctx context.Context, fi *dokan.FileInfo, buff []byte, offset int64) (n int, err error) {
		self.WriteLock()
		defer self.WriteUnLock()
		defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovered in WriteFile", r)
			log.Printf("%s: %s", r, debug.Stack())
			n = -1
			err = errors.New("Recovered in Writefile")
        }
    }()
	n=0

	path := toUnix(fi.Path())
	dbgcall("(WriteFile) Start: " + path)
	
	m, ok := fs.FileMeta.GetVal(path)
	if !ok {
		dbg("(WriteFile) Cannot write: file not loaded!" + path)
		panic("(WriteFile) Cannot write: file not loaded!" + path)
	}
	dbg("Loaded file from cache: " + path)
    m.Dirty = true
	data := m.Data


	maxl := len(data)
	dbg(fmt.Sprintf("(WriteFile) File of length %v (%v)", maxl, path))
	dbg(fmt.Sprintf("(WriteFile) Requested write at offset %v bytes", offset))

	if int(offset)+len(buff) > maxl {
		newmaxl := int(offset) + len(buff)
		needBytes := newmaxl-maxl
		data = append(data, make([]byte, newmaxl-maxl)...)
		dbg(fmt.Sprintf("(WriteFile) Tried to add %v bytes to resize file to %v bytes, actually got %v", needBytes, newmaxl, len(data)))
	}

	dataSlice := data[offset:]
	dbg(fmt.Sprintf("(WriteFile) Copying buffer of size %v into buffer of size %v, at offset %v", len(buff), len(dataSlice), offset))
	n = copy(dataSlice, buff)
	dbg(fmt.Sprintf("(WriteFile) Copied %v bytes to %v", n, offset) )

/*
	t := hashare.BeginTransaction(Conf())
	t, ok = hashare.DeleteFile(Conf().Store, path, Conf(),  t)
	if !ok {
		log.Println("(WriteFile) WriteFile: Couldn't delete:", path)
	}
	t, ok = hashare.PutBytes(Conf().Store, data, path, Conf(), true, t)
		if !ok {
		log.Println("(WriteFile) Couldn't PutBytes:", path)
		return 0, errors.New("(WriteFile) Couldn't write")
	}

	ok = hashare.CommitTransaction(t, "(WriteFile) Write to " + path, Conf())
	if !ok {
		log.Println("(WriteFile) Couldn't save:", path)
		return 0, errors.New("(WriteFile) Couldn't write")
	}
*/
	m.Data = data
	dbg("Storing file in cache: " + path)
	fs.FileMeta.SetVal(path, m)
	data = []byte{}
	dbg("(WriteFile) Finished writing: " + path)
    runtime.GC()

	dbg("(WriteFile) Finish" + path)
	return n, nil
}

func (t VortFile) FlushFileBuffers(ctx context.Context, fi *dokan.FileInfo) error {
	path := toUnix(fi.Path())
	dbgcall("VortFS.FlushFileBuffers " + path)
//	t.Flush(path)
	return nil
}


func (t VortFile) GetFileInformation(ctx context.Context, fi *dokan.FileInfo) (ds *dokan.Stat, err error) {
	defer func() {
        if r := recover(); r != nil {
			message := "Recovered in GetFileInformation"
            fmt.Println(message , r)
			log.Printf("%s: %s", r, debug.Stack())
			err = errors.New(message)
			ds = &dokan.Stat{}
        }
    }()
	//dbg("VortFile.GetFileInformation: " + fi.Path())
	var st dokan.Stat
	path := toUnix(fi.Path())
	f, ok := hashare.GetCurrentMeta(path, Conf())
	if !ok {
		return &st, errors.New("Could not find"+path)
	}
	dbg(fmt.Sprintf("GetFileInf: got meta: %+v", f))
	    // Timestamps for the file
    st.Creation = f.Modified
	st.LastAccess = f.Modified
	st.LastWrite = f.Modified
    // FileSize is the size of the file in bytes
    
    // FileIndex is a 64 bit (nearly) unique ID of the file
    st.FileIndex = 0
    // VolumeSerialNumber is the serial number of the volume (0 is fine)
    st.VolumeSerialNumber = 0
    // NumberOfLinks can be omitted, if zero set to 1.
    st.NumberOfLinks = 1
	
	if string(f.Type) == "dir" {
		st.FileAttributes = dokan.FileAttributeDirectory
		st.FileSize = 0
		dbg("This is a directory")
	} else {
		st.FileAttributes = dokan.FileAttributeNormal
		t, ok := fs.FileMeta.GetVal(path)
		
		if ok {
		data := t.Data
		if int64(len(data)) == 0 {
			st.FileSize = f.Size 
			dbg(fmt.Sprintf("Using filesize %v from metadata", st.FileSize))
		} else {
			st.FileSize = int64(len(data))
			dbg(fmt.Sprintf("Using filesize %v from data length", st.FileSize))
		}
		} else {
			st.FileSize = f.Size 
			dbg(fmt.Sprintf("Using filesize %v from metadata", st.FileSize))
		}
	}
	dbg("VortFile.GetFileInformation: " + fmt.Sprintf("%v, %v, %v, %v", path, st.FileSize, st.LastWrite, string(f.Type)))
	return &st, nil
}

func (t VortFile) FindFiles(ctx context.Context, fi *dokan.FileInfo, p string, cb func(*dokan.NamedStat) error) error {
	path := toUnix(fi.Path())
	dbgcall("VortFile.FindFiles "+ path)
	directory := path
	dbg(fmt.Sprintf("VortFile.FindFiles: Getting files for directory: %+v, filter: %v", directory, p))

	conf := Conf()

	dbg("VortFile.FindFiles")
	unixPath := strings.Replace(directory, "\\", "/", -1)
	files, _ := hashare.List(conf.Store, unixPath, conf)
	for _, f := range files {

		st := dokan.NamedStat{}
		st.Name = string(f.Name)
		st.NumberOfLinks = 1
		/*
			if len(string(f.Name)) > 8 {
				st.ShortName = string(f.Name)[0:7]
			} else {
				st.ShortName = string(f.Name)[0 : len(string(f.Name))-1]
			}
		*/
		st.FileSize = int64(f.Size)
		if st.FileSize < 0 {
			st.FileSize = 0
		}
		st.FileIndex = binary.LittleEndian.Uint64(f.Id)
		st.Creation = time.Now()
		st.LastWrite = time.Now()
		st.LastAccess = time.Now()

		if string(f.Type) == "dir" {
			st.FileAttributes = dokan.FileAttributeDirectory
		} else {
			st.FileAttributes = dokan.FileAttributeNormal
		}
		//log.Printf("findfiles returning struct: %+v", st)
		dbg(string(f.Name))
		cb(&st)
	}
	dbg(fmt.Sprintf("VortDir.FindFiles: Finished getting files for directory: %+v, filter: %v", directory, p))
	return nil
}
func (t VortFile) SetFileTime(cb context.Context, fi *dokan.FileInfo, t1 time.Time, t2 time.Time, t3 time.Time) error {
	dbgcall("VortFile.SetFileTime"+fi.Path())
	return nil
}
func (t VortFile) SetFileAttributes(ctx context.Context, fi *dokan.FileInfo, fileAttributes dokan.FileAttribute) error {
	dbgcall("VortFile.SetFileAttributes " + fi.Path())
	return nil
}

func (t VortFile) LockFile(ctx context.Context, fi *dokan.FileInfo, offset int64, length int64) error {
	dbgcall("VortFile.LockFile " + fi.Path())
	return nil
}
func (t VortFile) UnlockFile(ctx context.Context, fi *dokan.FileInfo, offset int64, length int64) error {
	dbgcall("VortFile.UnlockFile " + fi.Path() )
	return nil
}

var ccc *hashare.Config


type Config struct {
	Mount	string
	Repository string
	Username 	string
	Password	string
	UseCompression bool
}

func Conf() *hashare.Config {
	if ccc != nil {
		return ccc
	}
	fmt.Println("Contacting server for config")
	
	var userconfig Config
	raw, err := ioutil.ReadFile("vort.config")
	if err != nil {
		panic(err)
	}

	json.Unmarshal(raw, &userconfig)
	
	var conf = &hashare.Config{Debug: true, DoubleCheck: false, UserName: userconfig.Username, Password: userconfig.Password, Blocksize: 500, UseCompression: userconfig.UseCompression, UseEncryption: false, EncryptionKey: []byte("a very very very very secret key")} // 32 bytes
	
	//var s hashare.SiloStore
	store := hashare.AutoOpen(userconfig.Repository)
	
		log.Printf("Using conf %v", conf)
	if !hashare.Authenticate(store, conf) {
		fmt.Println("Could not log into the server.  Please check your username and password")
		os.Exit(1)
	}
	conf = hashare.Init(store, conf)
	conf.Debug = true
	fmt.Printf("Chose config: %+v\n", conf)
	//os.Exit(1)
	ccc = conf
	return conf
}

var fileTime time.Time
