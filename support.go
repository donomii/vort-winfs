package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"time"

	"github.com/donomii/hashare"
)

func SaveJsonToFile(filename string, data hashare.ClientConfig) {
	b, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}
	ioutil.WriteFile(filename, b, 0777)
}

func DeleteVal(m *hashare.FileCache, key string) {
	m.Delete(key)
}

func (self *VortFS) GetVal(m *hashare.FileCache, key string) (*hashare.VortFile, bool) {
	val, ok := m.GetVal(key)
	if ok {
		return val, ok
	} else {
		return nil, ok
	}
}

func (self *VortFS) SetVal(m *hashare.FileCache, key string, val interface{}) {
	m.SetVal(key, val)
}

func traceaction(s string) {
	if traceOutput {
		log.Println("(fusecall)" + s)
	}
}

func errlog(s string) {
	log.Println("(fuse)" + s)
}

func zeroBytes(b []byte) {
	for i, _ := range b {
		b[i] = 0
	}
}

func MemStatsPrinter() {
	go func() {
		for {
			//var m runtime.MemStats
			//runtime.ReadMemStats(&m)
			//log.Printf("\nAlloc = %v\nTotalAlloc = %v\nSys = %v\nNumGC =      %v\n\n", m.Alloc/1024, m.TotalAlloc/1024, m.Sys/1024, m.NumGC)
			time.Sleep(60 * time.Second)
		}
	}()
}
