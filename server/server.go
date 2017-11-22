package server

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strconv"
	"time"

	gofast "github.com/lvsiquan/gofast"
)

const LOGFILE_PRIFIX = "logmover_"

type log struct {
	From string
	Log  string
}

var port = flag.Int("p", 45456, "Port")

//Start server
func Start() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()

	udp_addr, err := net.ResolveUDPAddr("udp", ":"+strconv.Itoa(*port))
	if err != nil {
		return
	}

	conn, err := net.ListenUDP("udp", udp_addr)
	defer conn.Close()
	if err != nil {
		os.Exit(1)
	}

	os.Setenv("MAX_WORKER", "20")
	os.Setenv("MAX_QUEUE", "200")
	d := gofast.NewDispatcher()

	var fileMap map[string]*os.File = make(map[string]*os.File)

	go d.Run(func(job gofast.Job) {
		payload := job.Payload.(*log)
		now := time.Now()
		yesterday := now.AddDate(0, 0, -1)
		logfile := payload.From + "-" + LOGFILE_PRIFIX + now.Format("2006_01_02") + ".log"
		var file *os.File
		v, ok := fileMap[logfile]
		if ok {
			file = v
		} else {
			file, err := os.OpenFile(logfile, os.O_WRONLY|os.O_APPEND, 0666)

			if err != nil && os.IsNotExist(err) {
				f, err := os.Create(logfile)
				if err != nil {
					fmt.Println("create file [" + logfile + "] failed!!!")
					return
				}
				file = f
				yesLogfile := payload.From + "-" + LOGFILE_PRIFIX + yesterday.Format("2006_01_02") + ".log"
				yesFile, err := os.Open(yesLogfile)
				if err == nil {
					files := []*os.File{yesFile}
					go Compress(files, yesLogfile+".tar.gz")
					fileMap[yesLogfile].Close()
					delete(fileMap, yesLogfile)
				}

			}

			fileMap[logfile] = file

		}

		// defer file.Close()
		file.WriteString(payload.Log + "\n")

	})

	var ringBuffer = NewRingBuffer(200)

	go func(ringBuffer *RingBuffer) {
		for {
			if data, err := ringBuffer.Get(); err == nil {
				job := gofast.Job{Payload: data}
				gofast.JobQueue <- job
			} else {
				continue
			}
		}
	}(ringBuffer)

	for {
		var buf = make([]byte, 1024)
		n, from, err := conn.ReadFromUDP(buf[0:])
		if err != nil {
			fmt.Println("Read from UDP failed!!!")
			continue
		}
		if n > 0 {
			l := &log{From: from.IP.String(), Log: string(buf[:n])}
			ringBuffer.Put(l)
		}

	}

}

func Compress(files []*os.File, dest string) error {
	d, _ := os.Create(dest)
	defer d.Close()
	gw := gzip.NewWriter(d)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	for _, file := range files {
		err := compress(file, "", tw)
		if err != nil {
			return err
		}
	}
	return nil
}

func compress(file *os.File, prefix string, tw *tar.Writer) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		prefix = prefix + "/" + info.Name()
		fileInfos, err := file.Readdir(-1)
		if err != nil {
			return err
		}
		for _, fi := range fileInfos {
			f, err := os.Open(file.Name() + "/" + fi.Name())
			if err != nil {
				return err
			}
			err = compress(f, prefix, tw)
			if err != nil {
				return err
			}
		}
	} else {
		header, err := tar.FileInfoHeader(info, "")
		header.Name = prefix + "/" + header.Name
		if err != nil {
			return err
		}
		err = tw.WriteHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(tw, file)
		file.Close()
		os.Remove(file.Name())
		if err != nil {
			return err
		}
	}
	return nil
}
