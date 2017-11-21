package server

import (
	"flag"
	"fmt"
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
	go d.Run(func(job gofast.Job) {
		payload := job.Payload.(*log)
		now := time.Now()
		logfile := payload.From + "-" + LOGFILE_PRIFIX + now.Format("2006_01_02") + ".log"
		file, err := os.OpenFile(logfile, os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil && os.IsNotExist(err) {
			f, err := os.Create(logfile)
			if err != nil {
				fmt.Println("create file [" + logfile + "] failed!!!")
				return
			}
			file = f
		}
		defer file.Close()
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
