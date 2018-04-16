package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/donomii/hashare"

	"github.com/keybase/dokan-go"
	"github.com/keybase/kbfs/dokan/winacl"
	//"github.com/keybase/kbfs/ioutil"
	//"golang.org/x/sys/windows"
)

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
	fs := newTestFS()
	mnt, err := dokan.Mount(&dokan.Config{FileSystem: fs, Path: `T:\`, MountFlags: dokan.Removable})
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
	debug("emptyFS.Cleanup:" + fi.Path())
}

func (t emptyFile) CloseFile(ctx context.Context, fi *dokan.FileInfo) {
	debug("emptyFS.CloseFile: " + fi.Path())
}

func (t emptyFS) WithContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return ctx, nil
}

func (t emptyFS) GetVolumeInformation(ctx context.Context) (dokan.VolumeInformation, error) {
	debug("emptyFS.GetVolumeInformation")
	return dokan.VolumeInformation{FileSystemFlags: dokan.FileReadOnlyVolume, VolumeName: "Vort"}, nil
}

func (t emptyFS) GetDiskFreeSpace(ctx context.Context) (dokan.FreeSpace, error) {
	debug("emptyFS.GetDiskFreeSpace")
	return dokan.FreeSpace{}, nil
}

func (t emptyFS) ErrorPrint(err error) {
	//debug(err)
}

func (t emptyFS) CreateFile(ctx context.Context, fi *dokan.FileInfo, cd *dokan.CreateData) (dokan.File, bool, error) {
	debug("emptyFS.CreateFile")
	return emptyFile{}, true, nil
}
func (t emptyFile) CanDeleteFile(ctx context.Context, fi *dokan.FileInfo) error {
	return dokan.ErrAccessDenied
}
func (t emptyFile) CanDeleteDirectory(ctx context.Context, fi *dokan.FileInfo) error {
	return dokan.ErrAccessDenied
}
func (t emptyFile) SetEndOfFile(ctx context.Context, fi *dokan.FileInfo, length int64) error {
	debug("emptyFile.SetEndOfFile")
	return nil
}
func (t emptyFile) SetAllocationSize(ctx context.Context, fi *dokan.FileInfo, length int64) error {
	debug("emptyFile.SetAllocationSize")
	return nil
}
func (t emptyFS) MoveFile(ctx context.Context, src dokan.File, sourceFI *dokan.FileInfo, targetPath string, replaceExisting bool) error {
	debug("emptyFS.MoveFile")
	return nil
}
func (t emptyFile) ReadFile(ctx context.Context, fi *dokan.FileInfo, bs []byte, offset int64) (int, error) {
	return len(bs), nil
}
func (t emptyFile) WriteFile(ctx context.Context, fi *dokan.FileInfo, bs []byte, offset int64) (int, error) {
	return len(bs), nil
}
func (t emptyFile) FlushFileBuffers(ctx context.Context, fi *dokan.FileInfo) error {
	debug("emptyFS.FlushFileBuffers")
	return nil
}

type emptyFile struct{}

func (t emptyFile) GetFileInformation(ctx context.Context, fi *dokan.FileInfo) (*dokan.Stat, error) {
	debug("emptyFile.GetFileInformation: " + fi.Path())
	var st dokan.Stat
	st.FileAttributes = dokan.FileAttributeNormal
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
	ramFile *ramFile
}

func newTestFS() *testFS {
	var t testFS
	t.ramFile = newRAMFile()
	return &t
}

func (t *testFS) CreateFile(ctx context.Context, fi *dokan.FileInfo, cd *dokan.CreateData) (dokan.File, bool, error) {
	path := fi.Path()
	debug("testFS.CreateFile:" + path)
	var s hashare.SiloStore
	s = hashare.NewHttpStore(repository)
	conf = hashare.Init(s, conf)
	switch path {

	case `\`, ``:
		if cd.CreateOptions&dokan.FileNonDirectoryFile != 0 {
			return nil, true, dokan.ErrFileIsADirectory
		}
		return testDir{}, true, nil
	default:
		unixPath := strings.Replace(path, "\\", "/", -1)
		f, ok := hashare.GetMeta(s, unixPath, conf)
		if ok {
			log.Printf("Got metadata: %+v", f)
			if string(f.Type) == "dir" {
				log.Println("Returning dir for:", unixPath)
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
	return nil, false, dokan.ErrObjectNameNotFound
}
func (t *testFS) GetDiskFreeSpace(ctx context.Context) (dokan.FreeSpace, error) {
	debug("testFS.GetDiskFreeSpace")
	return dokan.FreeSpace{
		FreeBytesAvailable:     testFreeAvail,
		TotalNumberOfBytes:     testTotalBytes,
		TotalNumberOfFreeBytes: testTotalFree,
	}, nil
}

const (
	// Windows mangles the last bytes of GetDiskFreeSpaceEx
	// because of GetDiskFreeSpace and sectors...
	testFreeAvail  = 0xA234567887654000
	testTotalBytes = 0xB234567887654000
	testTotalFree  = 0xC234567887654000
)

type testDir struct {
	emptyFile
}

var conf = &hashare.Config{Debug: false, DoubleCheck: false, UserName: "abcd", Password: "efgh", Blocksize: 500, UseCompression: true, UseEncryption: false, EncryptionKey: []byte("a very very very very secret key")} // 32 bytes
var repository = "http://192.168.1.101:80/"

func (t testDir) FindFiles(ctx context.Context, fi *dokan.FileInfo, p string, cb func(*dokan.NamedStat) error) error {
	directory := fi.Path()
	log.Printf("Getting files for directory: %+v", directory)
	var s hashare.SiloStore
	s = hashare.NewHttpStore(repository)
	conf = hashare.Init(s, conf)

	debug("testDir.FindFiles")
	unixPath := strings.Replace(directory, "\\", "/", -1)
	files, _ := hashare.List(s, unixPath, conf)
	for _, f := range files {

		st := dokan.NamedStat{}
		st.Name = string(f.Name)
		/*
			if len(string(f.Name)) > 8 {
				st.ShortName = string(f.Name)[0:7]
			} else {
				st.ShortName = string(f.Name)[0 : len(string(f.Name))-1]
			}
		*/
		st.FileSize = int64(f.Size)
		i, _ := strconv.ParseInt(string(f.Id), 10, 64)
		st.FileIndex = uint64(i)
		if string(f.Type) == "dir" {
			st.FileAttributes = dokan.FileAttributeDirectory
		} else {
			st.FileAttributes = dokan.FileAttributeNormal
		}
		cb(&st)
	}
	return nil
}
func (t testDir) GetFileInformation(ctx context.Context, fi *dokan.FileInfo) (*dokan.Stat, error) {
	debug("testDir.GetFileInformation" + fi.Path())
	return &dokan.Stat{
		FileAttributes: dokan.FileAttributeDirectory,
		FileSize:       int64(10),
		LastAccess:     time.Now(),
		LastWrite:      time.Now(),
		Creation:       time.Now(),
	}, nil
}

type testFile struct {
	emptyFile
}

func (t testFile) GetFileInformation(ctx context.Context, fi *dokan.FileInfo) (*dokan.Stat, error) {
	debug("Starting testFile.GetFileInformation" + fi.Path())
	unixPath := strings.Replace(fi.Path(), "\\", "/", -1)
	f, _ := hashare.GetMeta(conf.Store, unixPath, conf)
	debug(fmt.Sprintf("testFile.GetFileInformation: %v, %v", unixPath, f.Size))
	return &dokan.Stat{
		FileAttributes: dokan.FileAttributeNormal,
		FileSize:       int64(f.Size),
		LastAccess:     time.Now(),
		LastWrite:      time.Now(),
		Creation:       time.Now(),
	}, nil
}
func (t testFile) ReadFile(ctx context.Context, fi *dokan.FileInfo, bs []byte, offset int64) (int, error) {
	finish := offset + int64(len(bs))
	debug(fmt.Sprintf("ReadFile: %v - %v, %v", offset, finish, fi.Path()))
	data, _ := hashare.GetFile(conf.Store, fi.Path(), 0, -1, conf)
	debug(fmt.Sprintf("ReadFile loaded data: %v bytes, %v", len(data), fi.Path()))
	if offset >= int64(len(data)) {
		debug(fmt.Sprintf("Caught read at end of file, returning 0"))
		return 0, nil
	}
	if finish > int64(len(data)) {
		debug(fmt.Sprintf("Caught read past end of file, reducing end of read from %v to %v", finish, len(data)))
		finish = int64(len(data))
	}
	//debug("Got data " + string(data))
	//rd := bytes.NewReader(data)
	debug(fmt.Sprintf("ReadFile copying: %v - %v, %v", offset, finish, fi.Path()))
	count := copy(bs, data[offset:finish])
	debug(fmt.Sprintf("ReadFile: copied %v bytes,  %v - %v", count, offset, finish))

	return count, nil
}

type ramFile struct {
	emptyFile
	lock          sync.Mutex
	creationTime  time.Time
	lastReadTime  time.Time
	lastWriteTime time.Time
	contents      []byte
}

func newRAMFile() *ramFile {
	var r ramFile
	r.creationTime = time.Now()
	r.lastReadTime = r.creationTime
	r.lastWriteTime = r.creationTime
	return &r
}

func (r *ramFile) GetFileInformation(ctx context.Context, fi *dokan.FileInfo) (*dokan.Stat, error) {
	debug("ramFile.GetFileInformation")
	r.lock.Lock()
	defer r.lock.Unlock()
	return &dokan.Stat{
		FileSize:   int64(len(r.contents)),
		LastAccess: r.lastReadTime,
		LastWrite:  r.lastWriteTime,
		Creation:   r.creationTime,
	}, nil
}

func (r *ramFile) ReadFile(ctx context.Context, fi *dokan.FileInfo, bs []byte, offset int64) (int, error) {
	debug("ramFile.ReadFile")
	r.lock.Lock()
	defer r.lock.Unlock()
	r.lastReadTime = time.Now()
	rd := bytes.NewReader(r.contents)
	return rd.ReadAt(bs, offset)
}

func (r *ramFile) WriteFile(ctx context.Context, fi *dokan.FileInfo, bs []byte, offset int64) (int, error) {
	debug("ramFile.WriteFile")
	r.lock.Lock()
	defer r.lock.Unlock()
	r.lastWriteTime = time.Now()
	maxl := len(r.contents)
	if int(offset)+len(bs) > maxl {
		maxl = int(offset) + len(bs)
		r.contents = append(r.contents, make([]byte, maxl-len(r.contents))...)
	}
	n := copy(r.contents[int(offset):], bs)
	return n, nil
}
func (r *ramFile) SetFileTime(ctx context.Context, fi *dokan.FileInfo, creationTime time.Time, lastReadTime time.Time, lastWriteTime time.Time) error {
	debug("ramFile.SetFileTime")
	r.lock.Lock()
	defer r.lock.Unlock()
	if !lastWriteTime.IsZero() {
		r.lastWriteTime = lastWriteTime
	}
	return nil
}
func (r *ramFile) SetEndOfFile(ctx context.Context, fi *dokan.FileInfo, length int64) error {
	debug("ramFile.SetEndOfFile")
	r.lock.Lock()
	defer r.lock.Unlock()
	r.lastWriteTime = time.Now()
	switch {
	case int(length) < len(r.contents):
		r.contents = r.contents[:int(length)]
	case int(length) > len(r.contents):
		r.contents = append(r.contents, make([]byte, int(length)-len(r.contents))...)
	}
	return nil
}
func (r *ramFile) SetAllocationSize(ctx context.Context, fi *dokan.FileInfo, length int64) error {
	debug("ramFile.SetAllocationSize")
	r.lock.Lock()
	defer r.lock.Unlock()
	r.lastWriteTime = time.Now()
	switch {
	case int(length) < len(r.contents):
		r.contents = r.contents[:int(length)]
	}
	return nil
}
