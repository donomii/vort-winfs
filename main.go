package main

import (
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

	"github.com/donomii/hashare"

	"github.com/keybase/dokan-go"
	"github.com/keybase/kbfs/dokan/winacl"
	//"github.com/keybase/kbfs/ioutil"
	//"golang.org/x/sys/windows"
)


type VortFile struct {
	hashare.VortFile
	}
var fs VortFS

//Writing a filesystem

// If you're interested in writing your own filesystem (and driver), then you've probably
// done some research on it and given up, because it looks incredibly hard.  That has changed
// now, because dokany makes it quite easy.  However dokany doesn't come with any tutorial,
// so it assumes that you already know how to do the thing you're trying to do.

// Dokany-go also doesn't have much in the way of docs, so I'll write a quick summary of how
// I got this going, as I go along.  Hopefully it will help a few others along the way.
//
// To start, grab the template file from the examples directory, and get it to the point
// where you can compile and run it.  If the example doesn't run, nothing in here will matter.

// Dokany works by calling functions that you write.  These calls happen when the OS wants
// to list a directory, open a file, etc.  You will be writing the calls that are normally
// provided by the operating system, like CreateFile().

// The best place to start is to fill out FindFiles().  This will allow you to do directory
// listings.

// The pattern is fairly simple:  You are going to get a list of files from somewhere, and
// call a callback function on each item in that list.

/*
files := GetMyFiles(fi.Path())
	for _, f := range files {

		st := dokan.NamedStat{}
		st.Name = string(f.Name)
		st.FileSize = int64(f.Size)

		if f.Type == "dir" {
			st.FileAttributes = dokan.FileAttributeDirectory
		} else {
			st.FileAttributes = dokan.FileAttributeNormal
		}
		cb(&st)
	}
*/

// This is very straightforward.  Each file needs a name, a size, and a "FileAttributes",
// which indicates if it is a directory, file, or "special" (e.g. a shortcut).  We won't be
// looking at special files here.

// Compile, run, and now you can see a file list in your drive.  Not bad for only 10 minutes work.

// However, you will notice that this doesn't work for subdirectories.  It turns out that
// windows doesn't trust your directory listing, so when you try to list the files in a
// directory, windows will attempt to open that directory with CreateFile(), and then decide
// what to do based on the result of that call.  So we need to fill out CreateFile() too.

// This is also simple, so let's add a default: option to the case statement in CreateFile()

/*
default:
		f, ok := GetMetaData(fi.Path())
		if ok {
			log.Printf("Got metadata: %+v", f)
			if f.Type == "dir" {
				log.Println("Returning dir for:", fi.Path())
				return VortDir{}, true, nil
			}

			log.Println("Returning file for:", fi.Path())
			return testFile{}, false, nil
		}
*/

// There are a lot more options to consider here, so we will revisit this later.

// CreateFile is what windows calls when it wants to open a file.  "Open()" was already
// taken by Unix, and Microsoft has always been desperate to do the same thing differently,
// so we end up with silliness like this.

// In any case, once CreateFile has been called, the file is considered "open", and the
// operating system can read and write to it.  Let's start with reading.

// Add your code to ReadFile()

/*
	data := GetFile(fi.Path())
	rd := bytes.NewReader(data)
	return rd.ReadAt(bs, offset)
*/

// And that's it!  You now have a minimal working filesystem!  Users can mount it, and read files from it.
// It took me less than 3 hours, and most of that was setting up the template.  I could do it in 30 now.

// So what are all the other functions for?

// Caveats

// Some slightly amusing, but mostly frustrating notes:
//
// Because windows will consider this mount to be a local drive, even if you are actually writing a network
// filesystem, windows will do things like attempt to create a thumbnail for every picture in every directory
// you look at.  So if you have a large pictures directory on your network fileshare, windows will pull gigs
// across the network and then throw it all away.  I haven't figured out how to mark this as a network drive
// to prevent this behaviour.

func main() {
	fmt.Println("vort-winfs started")
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
	
	fs = VortFS{FileMeta: map[string]*VortFile{}}

	
	Conf()
	mnt, err := dokan.Mount(&dokan.Config{FileSystem: &fs, Path: drive, MountFlags: dokan.Network | dokan.Removable})
	if err != nil {
		log.Fatal("Mount failed:", err)
	}
	log.Println("Mount successful, ready to serve")
	for {
		time.Sleep(1 * time.Second)
	}
	defer mnt.Close()
}

var _ dokan.FileSystem = VortFS{}

type VortFS struct{
	NextFileHandle	uint64
	Config		*hashare.Config
    FileMeta    map[string]*VortFile
}

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
}

func (t VortFile) CloseFile(ctx context.Context, fi *dokan.FileInfo) {
	dbg("VortFS.CloseFile: " + fi.Path())
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
        m := VortFile{}
        m.Data = make([]byte, size)
        m.Loaded = true
		m.Dirty = true
		self.FileMeta[path] = &m
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
	defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovered in Flush", r)
			log.Printf("%s: %s", r, debug.Stack()) 
			errc = -1
        }
    }()
	dbgcall("Flush " + path)
	conf := Conf()
	
	meta, ok := self.FileMeta[path] 
    if ok && meta.Dirty {
        t := hashare.BeginTransaction(conf)
        t, ok = hashare.DeleteFile(conf.Store, path, conf, t)
		if !ok {
			dbg("Could not delete file in flush:"+path)
			//If the file doesn't exist, we don't fail, we just create it
			//return -fuse.EIO
		}
        t, ok = hashare.PutBytes(conf.Store, meta.Data, path, conf, true, t)
		if !ok {
			dbg("Could not put file in flush:"+path)
			return -1
		}
        hashare.CommitTransaction(t, "Flush " + path, conf)
    }
	dbg("Flushed " + path)
	return 0
}

func (self VortFS) CreateFile(ctx context.Context, fi *dokan.FileInfo, cd *dokan.CreateData) (file dokan.File, isDirectory bool, err error) {

	unixPath := strings.Replace(fi.Path(), "\\", "/", -1)
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
	
	dbgcall(fmt.Sprintf("Createfile %v, %v, %v", cd.CreateOptions, cd.CreateDisposition, path ))

	switch path {
	case `\`, ``:
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
			case dokan.CreateDisposition(dokan.FileOpenIf):
				dbg("Createfile: Conditional open:"+unixPath)
			case dokan.CreateDisposition(dokan.FileSupersede):
				dbg("Createfile: Supersede:"+unixPath)
			case dokan.CreateDisposition(dokan.FileOverwrite):
				dbg("Createfile: Overwrite:"+unixPath)
			case dokan.CreateDisposition(dokan.FileOverwriteIf):
				dbg("Createfile: Conditional overwrite:"+unixPath)
			case dokan.CreateDisposition(dokan.FileCreate):
				dbg("Createfile: Creating empty file:"+unixPath)
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
			return VortFile{}, false, nil
		}
		log.Println("Returning file for:", unixPath)
		dbg("Returning file for: "+ unixPath)
		self.FileMeta[path] = &VortFile{}
		if !fs.FileMeta[path].Loaded {
			fs.LoadFile(path)
		}
		dbg("Opened " + path + ", FH is " + fmt.Sprintf("%v",self.NextFileHandle))
		return VortFile{}, false, nil
		}
		


	return
}
func (t VortFile) CanDeleteFile(ctx context.Context, fi *dokan.FileInfo) error {
	dbgcall("CanDeleteFile " + fi.Path() )
	return nil
}

func (t VortFile) CanDeleteDirectory(ctx context.Context, fi *dokan.FileInfo) error {
	dbgcall("CanDeleteDirectory " + fi.Path() )
	return nil
}
func (self VortFile) SetEndOfFile(ctx context.Context, fi *dokan.FileInfo, length int64) (err error) {
	defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovered in setendoffile", r)
			log.Printf("%s: %s", r, debug.Stack()) 
			err = errors.New("Recovered int setendoffile")
        }
    }()
	
	path := fi.Path()
	dbgcall("Truncate " + path + " to " + fmt.Sprintf("%v", length))
	
	dbg("VortFile.SetEndOfFile Start" + fi.Path())
	if !fs.FileMeta[path].Loaded {
        fs.LoadFile(path)
	}
	
	data := fs.FileMeta[path].Data
	switch {
	case int(length) < len(data):
		data = data[:int(length)]
	case int(length) > len(data):
		data = append(data, make([]byte, int(length)-len(data))...)
	}
	
	
	fs.FileMeta[path].Data = data
	
	dbg("VortFile.SetEndOfFile Finish" + path)
	
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
	dbg("VortFile.SetAllocationSize Start" + fi.Path())
	data, ok := hashare.GetFile(Conf().Store, fi.Path(), 0, -1, Conf())
	if !ok {
		data = []byte{}
	}
	switch {
	case int(length) < len(data):
		data = data[:int(length)]
	}
	transaction := hashare.BeginTransaction(Conf())
	transaction, ok = hashare.DeleteFile(Conf().Store, fi.Path(), Conf(),  transaction)
	transaction, ok = hashare.PutBytes(Conf().Store, data, fi.Path(), Conf(), true, transaction)
	hashare.CommitTransaction(transaction, "Set allocation size", Conf())
	if !ok {
		log.Println("SetAllocatiionSize: Couldn't save:", fi.Path())
		return errors.New("Couldn't write")
	}
	dbg("VortFile.SetAllocationSize Finish" + fi.Path())
	return nil
}

func (t VortFS) MoveFile(ctx context.Context, src dokan.File, sourceFI *dokan.FileInfo, targetPath string, replaceExisting bool) error {
	dbgcall("VortFS.MoveFile "+sourceFI.Path())
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
	err = nil
	dbgcall(fmt.Sprintf("ReadFile %v. Want %v bytes from offset %v", fi.Path(), len(bs), offset  ))
	conf := Conf()
	start := offset
	finish := offset + int64(len(bs))
	//dbg(fmt.Sprintf("ReadFile: %v - %v, %v", offset, finish, fi.Path()))
	log.Printf("ReadFile: %v - %v, %v", offset, finish, fi.Path())
	data, ok := hashare.GetFile(conf.Store, fi.Path(), 0, -1, conf)
	if !ok {
		log.Println("File not found:", fi.Path())
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
	dbg(fmt.Sprintf("ReadFile Complete: %v - %v, %v", offset, finish, fi.Path()))
	count := copy(bs, data[offset:finish])
	n = count
	dbg(fmt.Sprintf("ReadFile: copied %v bytes,  %v - %v, with error %v", count, offset, finish, err))
	return count, err
}


func (self *VortFS) LoadFile(path string) {
		dbg("LoadFile "+path)
        data, ok := hashare.GetFile(Conf().Store, path, 0, -1, Conf())
		if !ok {
			fmt.Println("Loadfile failed!")
		}
        m := VortFile{}
        m.Data = data
        m.Loaded = true
		self.FileMeta[path] = &m
		dbg("LoadFile complete: "+path)
}

func (self VortFile) CheckLoaded(path string) {
	_, ok := fs.FileMeta[path]
	
	if !(ok && fs.FileMeta[path].Loaded) {
        fs.LoadFile(path)
	}
}

func (self VortFile) WriteFile(ctx context.Context, fi *dokan.FileInfo, buff []byte, offset int64) (n int, err error) {
		defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovered in WriteFile", r)
			log.Printf("%s: %s", r, debug.Stack())
			n = -1
			err = errors.New("Recovered in Writefile")
        }
    }()
	n=0

	path := fi.Path()
	dbg("VortFile.WriteFile Start" + path)
	self.CheckLoaded(path)
	time.Sleep(1*time.Second)
	
	
    m, ok := fs.FileMeta[path]
	if !ok {
		dbg("Cannot write: file not loaded!")
	}
    m.Dirty = true
	data := m.Data


	maxl := len(data)
	dbg("Fetched file of length " + fmt.Sprintf("%v", maxl))
	dbg(fmt.Sprintf("Requested write at offset %v bytes", offset))

	if int(offset)+len(buff) > maxl {
		newmaxl := int(offset) + len(buff)
		needBytes := newmaxl-maxl
		data = append(data, make([]byte, newmaxl-maxl)...)
		dbg(fmt.Sprintf("Tried to add %v bytes to resize file to %v bytes, actually got %v", needBytes, newmaxl, len(data)))
	}

	dataSlice := data[offset:]
	dbg(fmt.Sprintf("Copying buffer of size %v into buffer of size %v, at offset %v", len(buff), len(dataSlice), offset))
	n = copy(dataSlice, buff)
	dbg(fmt.Sprintf("Copied %v bytes to %v", n, offset) )


	t := hashare.BeginTransaction(Conf())
	t, ok = hashare.DeleteFile(Conf().Store, fi.Path(), Conf(),  t)
	t, ok = hashare.PutBytes(Conf().Store, data, fi.Path(), Conf(), true, t)
		if !ok {
		log.Println("WriteFile: Couldn't save:", fi.Path())
		return 0, errors.New("Couldn't write")
	}

	ok = hashare.CommitTransaction(t, "Write to " + fi.Path(), Conf())
	if !ok {
		log.Println("WriteFile: Couldn't save:", fi.Path())
		return 0, errors.New("Couldn't write")
	}

	m.Data = data
	data = []byte{}
	dbg("Finished writing: " + path)
    runtime.GC()

	dbg("VortFile.WriteFile Finish" + fi.Path())
	return n, nil
}

func (t VortFile) FlushFileBuffers(ctx context.Context, fi *dokan.FileInfo) error {
	dbgcall("VortFS.FlushFileBuffers " +fi.Path())
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
	path := fi.Path()
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
		t, ok := fs.FileMeta[fi.Path()]
		
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
	dbgcall("VortFile.FindFiles "+ fi.Path())
	directory := fi.Path()
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
	/*
		st := dokan.NamedStat{}
		st.Name = "hello.txt"
		st.FileSize = int64(len(helloStr))

		cb(&st)
	*/
	dbg(fmt.Sprintf("VortDir.FileFiles: Finished getting files for directory: %+v, filter: %v", directory, p))
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
	
	var conf = &hashare.Config{Debug: true, DoubleCheck: false, UserName: userconfig.Username, Password: userconfig.Password, Blocksize: 500, UseCompression: true, UseEncryption: false, EncryptionKey: []byte("a very very very very secret key")} // 32 bytes
	var s hashare.SiloStore
	store := hashare.NewHttpStore(userconfig.Repository)
	wc := hashare.NewWriteCacheStore(store)
	s = hashare.NewReadCacheStore(wc)
	conf = hashare.Init(s, conf)
	fmt.Printf("Chose config: %+v\n", conf)
	//os.Exit(1)
	ccc = conf
	return conf
}

var fileTime time.Time
