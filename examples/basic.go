package main

import (
	"bytes"
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/keybase/dokan-go"
	"github.com/keybase/kbfs/dokan/winacl"
	//"github.com/keybase/kbfs/ioutil"
	//"golang.org/x/sys/windows"
)

func main() {
	fs := newTestFS()
	mnt, err := dokan.Mount(&dokan.Config{FileSystem: fs, Path: `T:\`})
	if err != nil {
		log.Fatal("Mount failed:", err)
	}
	time.Sleep(500 * time.Second)
	defer mnt.Close()
}

var _ dokan.FileSystem = emptyFS{}

type emptyFS struct{}

func debug(s string) {}

func (t emptyFile) GetFileSecurity(ctx context.Context, fi *dokan.FileInfo, si winacl.SecurityInformation, sd *winacl.SecurityDescriptor) error {
	debug("emptyFS.GetFileSecurity")
	return nil
}
func (t emptyFile) SetFileSecurity(ctx context.Context, fi *dokan.FileInfo, si winacl.SecurityInformation, sd *winacl.SecurityDescriptor) error {
	debug("emptyFS.SetFileSecurity")
	return nil
}
func (t emptyFile) Cleanup(ctx context.Context, fi *dokan.FileInfo) {
	debug("emptyFS.Cleanup")
}

func (t emptyFile) CloseFile(ctx context.Context, fi *dokan.FileInfo) {
	debug("emptyFS.CloseFile")
}

func (t emptyFS) WithContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return ctx, nil
}

func (t emptyFS) GetVolumeInformation(ctx context.Context) (dokan.VolumeInformation, error) {
	debug("emptyFS.GetVolumeInformation")
	return dokan.VolumeInformation{}, nil
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
	debug("emptyFile.GetFileInformation")
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
	//debug("testFS.CreateFile" , path)
	switch path {
	case `\hello.txt`:
		return testFile{}, false, nil
	case `\ram.txt`:
		return t.ramFile, false, nil
	// SL_OPEN_TARGET_DIRECTORY may get empty paths...
	case `\`, ``:
		if cd.CreateOptions&dokan.FileNonDirectoryFile != 0 {
			return nil, true, dokan.ErrFileIsADirectory
		}
		return testDir{}, true, nil
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

const helloStr = "hello world\r\n"

func (t testDir) FindFiles(ctx context.Context, fi *dokan.FileInfo, p string, cb func(*dokan.NamedStat) error) error {
	debug("testDir.FindFiles")
	st := dokan.NamedStat{}
	st.Name = "hello.txt"
	st.FileSize = int64(len(helloStr))
	return cb(&st)
}
func (t testDir) GetFileInformation(ctx context.Context, fi *dokan.FileInfo) (*dokan.Stat, error) {
	debug("testDir.GetFileInformation")
	return &dokan.Stat{
		FileAttributes: dokan.FileAttributeDirectory,
	}, nil
}

type testFile struct {
	emptyFile
}

func (t testFile) GetFileInformation(ctx context.Context, fi *dokan.FileInfo) (*dokan.Stat, error) {
	debug("testFile.GetFileInformation")
	return &dokan.Stat{
		FileSize: int64(len(helloStr)),
	}, nil
}
func (t testFile) ReadFile(ctx context.Context, fi *dokan.FileInfo, bs []byte, offset int64) (int, error) {
	debug("testFile.ReadFile")
	rd := strings.NewReader(helloStr)
	return rd.ReadAt(bs, offset)
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
