package main

import(
	"fmt"
	"sync"
	"time"
	"io/ioutil"
	
	"encoding/json"
	"github.com/donomii/hashare"
	"github.com/donomii/dokan-go"
	"log"
)

var(
	traceOutput bool
)
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
	mnt, err := dokan.Mount(&dokan.Config{FileSystem: &fs, Path: drive, MountFlags: dokan.CDebug | dokan.CStderr | dokan.MountManager | dokan.Removable})
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
