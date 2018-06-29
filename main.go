package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
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

const helloStr = "hello world\r\n"

var repository = "http://192.168.1.101:80/"

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
				return testDir{}, true, nil
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
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			log.Printf("\nAlloc = %v\nTotalAlloc = %v\nSys = %v\nNumGC = %v\n\n", m.Alloc/1024, m.TotalAlloc/1024, m.Sys/1024, m.NumGC)
			time.Sleep(1 * time.Second)
		}
	}()
	drive := os.Args[1]
	repository = os.Args[2]
	fmt.Println("Attempting to mount", repository, "on", drive)
	fs := newTestFS()

	Conf()
	mnt, err := dokan.Mount(&dokan.Config{FileSystem: fs, Path: drive, MountFlags: dokan.Network | dokan.Removable})
	if err != nil {
		log.Fatal("Mount failed:", err)
	}
	log.Println("Mount successful, ready to serve")
	for {
		time.Sleep(1 * time.Second)
	}
	defer mnt.Close()
}

var _ dokan.FileSystem = emptyFS{}

type emptyFS struct{}

func debug(s string) {

	fmt.Println(s)
}

func (t emptyFile) GetFileSecurity(ctx context.Context, fi *dokan.FileInfo, si winacl.SecurityInformation, sd *winacl.SecurityDescriptor) error {
	debug("emptyFS.GetFileSecurity")
	return nil
}
func (t emptyFile) SetFileSecurity(ctx context.Context, fi *dokan.FileInfo, si winacl.SecurityInformation, sd *winacl.SecurityDescriptor) error {
	debug("emptyFS.SetFileSecurity")
	return nil
}
func (t emptyFile) Cleanup(ctx context.Context, fi *dokan.FileInfo) {
	//debug("emptyFS.Cleanup:" + fi.Path())
}

func (t emptyFile) CloseFile(ctx context.Context, fi *dokan.FileInfo) {
	//debug("emptyFS.CloseFile: " + fi.Path())

}

func (t emptyFS) WithContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return ctx, nil
}

func (t emptyFS) GetVolumeInformation(ctx context.Context) (dokan.VolumeInformation, error) {
	//debug("emptyFS.GetVolumeInformation")
	return dokan.VolumeInformation{
		VolumeName:             "VORT",
		MaximumComponentLength: 0xFF, // This can be changed.
		FileSystemFlags: dokan.FileCasePreservedNames | dokan.FileCaseSensitiveSearch |
			dokan.FileUnicodeOnDisk | dokan.FileSupportsRemoteStorage,
		//| dokan.FileSequentalWriteOnce
		FileSystemName: "VORT",
	}, nil
}

var dummyFreeSpace uint64 = 512 * 1024 * 1024 * 1024

func freeSpace() dokan.FreeSpace {
	return dokan.FreeSpace{
		TotalNumberOfBytes:     dummyFreeSpace * 4,
		TotalNumberOfFreeBytes: dummyFreeSpace * 3,
		FreeBytesAvailable:     dummyFreeSpace * 2,
	}

}
func (t emptyFS) GetDiskFreeSpace(ctx context.Context) (dokan.FreeSpace, error) {
	debug("emptyFS.GetDiskFreeSpace")
	return freeSpace(), nil
}

func (t emptyFS) ErrorPrint(err error) {
	//debug(err)
}

func (t emptyFS) CreateFile(ctx context.Context, fi *dokan.FileInfo, cd *dokan.CreateData) (dokan.File, bool, error) {
	debug("emptyFS.CreateFile")
	unixPath := strings.Replace(fi.Path(), "\\", "/", -1)
	f, ok := hashare.GetCurrentMeta(unixPath, Conf())

	if !ok {
		log.Println("File not found:", fi.Path())
		return emptyFile{}, false, dokan.ErrObjectNameNotFound
	}
	return emptyFile{int(f.Size)}, true, nil
}
func (t emptyFile) CanDeleteFile(ctx context.Context, fi *dokan.FileInfo) error {
	return nil
}
func (t emptyFile) CanDeleteDirectory(ctx context.Context, fi *dokan.FileInfo) error {
	return nil
}
func (t emptyFile) SetEndOfFile(ctx context.Context, fi *dokan.FileInfo, length int64) error {
	debug("emptyFile.SetEndOfFile Start" + fi.Path())
	return nil
	data, _ := hashare.GetFile(Conf().Store, fi.Path(), 0, -1, Conf())
	switch {
	case int(length) < len(data):
		data = data[:int(length)]
	case int(length) > len(data):
		data = append(data, make([]byte, int(length)-len(data))...)
	}
	transaction := hashare.BeginTransaction(Conf())
	transaction, ok := hashare.DeleteFile(Conf().Store, fi.Path(), Conf(), false, transaction)
	transaction, ok = hashare.PutBytes(Conf().Store, data, fi.Path(), Conf(), true, transaction)
	hashare.CommitTransaction(transaction, Conf())
	if !ok {
		log.Println("SetEndOfFile: Couldn't save:", fi.Path())
		return errors.New("Couldn't write")
	}
	debug("emptyFile.SetEndOfFile Finish" + fi.Path())
	t.Length = int(length)
	return nil
}
func (t emptyFile) SetAllocationSize(ctx context.Context, fi *dokan.FileInfo, length int64) error {
	debug("emptyFile.SetAllocationSize Start" + fi.Path())
	data, ok := hashare.GetFile(Conf().Store, fi.Path(), 0, -1, Conf())
	if !ok {
		data = []byte{}
	}
	switch {
	case int(length) < len(data):
		data = data[:int(length)]
	}
	transaction := hashare.BeginTransaction(Conf())
	transaction, ok = hashare.DeleteFile(Conf().Store, fi.Path(), Conf(), false, transaction)
	transaction, ok = hashare.PutBytes(Conf().Store, data, fi.Path(), Conf(), true, transaction)
	hashare.CommitTransaction(transaction, Conf())
	if !ok {
		log.Println("SetAllocatiionSize: Couldn't save:", fi.Path())
		return errors.New("Couldn't write")
	}
	debug("emptyFile.SetAllocationSize Finish" + fi.Path())
	t.Length = int(length)
	return nil
}
func (t emptyFS) MoveFile(ctx context.Context, src dokan.File, sourceFI *dokan.FileInfo, targetPath string, replaceExisting bool) error {
	debug("emptyFS.MoveFile")
	return nil
}
func (t emptyFile) ReadFile(ctx context.Context, fi *dokan.FileInfo, bs []byte, offset int64) (int, error) {
	panic("nope")
	return len(bs), nil
}
func (r emptyFile) WriteFile(ctx context.Context, fi *dokan.FileInfo, bs []byte, offset int64) (int, error) {
	debug("empty.WriteFile Start" + fi.Path())
	data, ok := hashare.GetFile(Conf().Store, fi.Path(), 0, -1, Conf())
	if !ok {
		data = []byte{}
	}

	maxl := len(data)

	if int(offset)+len(bs) > maxl {
		maxl = int(offset) + len(bs)
		data = append(data, make([]byte, maxl-len(data))...)
	}
	n := copy(data[int(offset):], bs)

	t := hashare.BeginTransaction(Conf())
	t, ok = hashare.DeleteFile(Conf().Store, fi.Path(), Conf(), false, t)
	t, ok = hashare.PutBytes(Conf().Store, data, fi.Path(), Conf(), true, t)
	hashare.CommitTransaction(t, Conf())
	if !ok {
		log.Println("WriteFile: Couldn't save:", fi.Path())
		return 0, errors.New("Couldn't write")
	}
	debug("empty.WriteFile Finish" + fi.Path())
	r.Length = maxl
	return n, nil
}

func (t emptyFile) FlushFileBuffers(ctx context.Context, fi *dokan.FileInfo) error {
	debug("emptyFS.FlushFileBuffers")
	return nil
}

type emptyFile struct {
	Length int
}

func (t emptyFile) GetFileInformation(ctx context.Context, fi *dokan.FileInfo) (*dokan.Stat, error) {
	debug("emptyFile.GetFileInformation: " + fi.Path())
	var st dokan.Stat
	st.FileSize = int64(t.Length)
	st.FileAttributes = dokan.FileAttributeNormal
	st.NumberOfLinks = 1
	return &st, nil
}
func (t emptyFile) FindFiles(context.Context, *dokan.FileInfo, string, func(*dokan.NamedStat) error) error {
	debug("emptyFile.FindFiles")
	return nil
}
func (t emptyFile) SetFileTime(context.Context, *dokan.FileInfo, time.Time, time.Time, time.Time) error {
	debug("emptyFile.SetFileTime")
	return nil
}
func (t emptyFile) SetFileAttributes(ctx context.Context, fi *dokan.FileInfo, fileAttributes dokan.FileAttribute) error {
	debug("emptyFile.SetFileAttributes")
	return nil
}

func (t emptyFile) LockFile(ctx context.Context, fi *dokan.FileInfo, offset int64, length int64) error {
	debug("emptyFile.LockFile")
	return nil
}
func (t emptyFile) UnlockFile(ctx context.Context, fi *dokan.FileInfo, offset int64, length int64) error {
	debug("emptyFile.UnlockFile")
	return nil
}

type testFS struct {
	emptyFS
}

func newTestFS() *testFS {
	var t testFS

	return &t
}

var ccc *hashare.Config

func Conf() *hashare.Config {
	if ccc != nil {
		return ccc
	}
	fmt.Println("Contacting server for config")
	var conf = &hashare.Config{Debug: true, DoubleCheck: false, UserName: "abcd", Password: "efgh", Blocksize: 500, UseCompression: true, UseEncryption: false, EncryptionKey: []byte("a very very very very secret key")} // 32 bytes
	var s hashare.SiloStore
	store := hashare.NewHttpStore(repository)
	wc := hashare.NewWriteCacheStore(store)
	s = hashare.NewReadCacheStore(wc)
	conf = hashare.Init(s, conf)
	fmt.Printf("Chose config: %+v\n", conf)
	//os.Exit(1)
	ccc = conf
	return conf
}

func (t *testFS) CreateFile(ctx context.Context, fi *dokan.FileInfo, cd *dokan.CreateData) (dokan.File, bool, error) {
	path := fi.Path()
	//debug("testFS.CrexateFile:" + path)

	switch path {
	case `\`, ``:
		if cd.CreateOptions&dokan.FileNonDirectoryFile != 0 {
			return nil, true, dokan.ErrFileIsADirectory
		}
		return testDir{}, true, nil
	default:
		unixPath := strings.Replace(path, "\\", "/", -1)
		f, ok := hashare.GetCurrentMeta(unixPath, Conf())
		if ok {
			log.Printf("Got metadata: %+v", f)
			if string(f.Type) == "dir" {
				log.Println("Returning dir for:", unixPath)
				if cd.CreateOptions&dokan.FileNonDirectoryFile != 0 {
					return nil, true, dokan.ErrFileIsADirectory
				}
				return testDir{}, true, nil
			}
			log.Println("Returning file for:", unixPath)
			return testFile{}, false, nil
		}
		/*
			files, _ := hashare.List(s, unixPath, conf)
			for _, f := range files {
				log.Println("Comparing", string(f.Name), "and", path)
				if string(f.Name) == path {
					if string(f.Type) == "dir" {
						log.Println("Returning dir for:", unixPath)
						return testDir{}, true, nil
					}

					log.Println("Returning file for:", unixPath)
					return testFile{}, false, nil
				}
			}
		*/
	}
	if cd.CreateDisposition == dokan.FileOpen {
		return nil, false, dokan.ErrObjectNameNotFound
	} else {
		//Create a new one
		t := hashare.BeginTransaction(Conf())
		_, _ = hashare.PutBytes(Conf().Store, []byte{}, fi.Path(), Conf(), true, t)
		hashare.CommitTransaction(t, Conf())
		return testFile{}, false, nil
	}
}
func (t *testFS) GetDiskFreeSpace(ctx context.Context) (dokan.FreeSpace, error) {
	debug("testFS.GetDiskFreeSpace")
	return freeSpace(), nil
}

type testDir struct {
	emptyFile
}

func (t testDir) FindFiles(ctx context.Context, fi *dokan.FileInfo, p string, cb func(*dokan.NamedStat) error) error {
	directory := fi.Path()
	debug(fmt.Sprintf("Getting files for directory: %+v, filter: %v", directory, p))

	conf := Conf()

	debug("testDir.FindFiles")
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
		cb(&st)
	}
	/*
		st := dokan.NamedStat{}
		st.Name = "hello.txt"
		st.FileSize = int64(len(helloStr))

		cb(&st)
	*/
	return nil
}

var fileTime time.Time

func (t testDir) GetFileInformation(ctx context.Context, fi *dokan.FileInfo) (*dokan.Stat, error) {
	//debug("testDir.GetFileInformation" + fi.Path())
	unixPath := strings.Replace(fi.Path(), "\\", "/", -1)
	f, ok := hashare.GetCurrentMeta(unixPath, Conf())

	if !ok {
		log.Println("File not found:", fi.Path())
		return &dokan.Stat{}, dokan.ErrObjectNameNotFound
	}
	//debug("GetFileInformation Complete: " + fi.Path())
	i, _ := strconv.ParseInt(string(f.Id), 10, 64)

	return &dokan.Stat{
		FileIndex:      uint64(i),
		Creation:       fileTime,
		LastAccess:     fileTime,
		LastWrite:      fileTime,
		FileAttributes: dokan.FileAttributeDirectory,
	}, nil
	return &dokan.Stat{}, nil
}

type testFile struct {
	emptyFile
}

func (t testFile) GetFileInformation(ctx context.Context, fi *dokan.FileInfo) (*dokan.Stat, error) {
	debug("testFile.GetFileInformation: " + fi.Path())
	unixPath := strings.Replace(fi.Path(), "\\", "/", -1)
	f, ok := hashare.GetCurrentMeta(unixPath, Conf())
	if !ok {
		log.Println("File not found:", fi.Path())
		return nil, dokan.ErrObjectNameNotFound
	}
	debug("GetFileInformation Complete: " + fi.Path())
	//i, _ := strconv.ParseInt(hashare.BytesToHex(f.Id), 10, 64)
	size := int64(f.Size)
	if size < 0 {
		size = 0
	}
	ret := &dokan.Stat{
		FileIndex:      binary.LittleEndian.Uint64(f.Id),
		FileSize:       size,
		Creation:       time.Now(),
		LastAccess:     time.Now(),
		LastWrite:      time.Now(),
		FileAttributes: dokan.FileAttributeNormal,
		NumberOfLinks:  1,
	}
	debug(fmt.Sprintf("File details: %+v", ret))
	return ret, nil
}
func (t testFile) ReadFile(ctx context.Context, fi *dokan.FileInfo, bs []byte, offset int64) (int, error) {
	var err error = nil
	conf := Conf()
	start := offset
	finish := offset + int64(len(bs))
	//debug(fmt.Sprintf("ReadFile: %v - %v, %v", offset, finish, fi.Path()))
	log.Printf("ReadFile: %v - %v, %v", offset, finish, fi.Path())
	data, ok := hashare.GetFile(conf.Store, fi.Path(), 0, -1, conf)
	if !ok {
		log.Println("File not found:", fi.Path())
		return 0, dokan.ErrObjectNameNotFound //FIXME different error types
	}
	if start >= int64(len(data)) {
		err = io.EOF
		debug(fmt.Sprintf("Caught read at end of file, returning 0"))
		return 0, err
	}
	if finish > int64(len(data)) {
		err = io.EOF
		debug(fmt.Sprintf("Caught read past end of file, reducing end of read from %v to %v", finish, len(data)))
		finish = int64(len(data))
	}
	if finish == int64(len(data)) {
		err = io.EOF
		debug(fmt.Sprintf("Caught read at end of file, reducing end of read from %v to %v", finish, len(data)))
	}
	//debug("Got data " + string(data))
	//rd := bytes.NewReader(data)
	//debug(fmt.Sprintf("ReadFile Complete: %v - %v, %v", offset, finish, fi.Path()))
	count := copy(bs, data[offset:finish])
	//debug(fmt.Sprintf("ReadFile: copied %v bytes,  %v - %v", count, offset, finish))
	return count, err
}
