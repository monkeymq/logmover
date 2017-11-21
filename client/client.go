package client

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/hpcloud/tail"
)

var server = flag.String("server", "192.168.1.1:45456", "Server address")

var logpath = flag.String("d", "/opt/xxx.log", "The path of the log file")

func Start() {
	flag.Parse()

	udp_addr, err := net.ResolveUDPAddr("udp", *server)
	if err != nil {
		fmt.Println("Can't connected to server")
		os.Exit(1)
	}
	conn, err := net.DialUDP("udp", nil, udp_addr)
	if err != nil {
		fmt.Println("Can't dial: ", err)
		os.Exit(1)
	}
	defer conn.Close()

	logfile := *logpath
	t, err := tail.TailFile(logfile, tail.Config{Follow: true})
	if err != nil {
		fmt.Println("Tailf [" + logfile + "] error")
		os.Exit(1)
	}

	for line := range t.Lines {
		sendLog(line.Text, conn)
	}
}

var logCache = make([]string, 5)
var maxSendTime = 3 * time.Second
var begin = time.Now()

func sendLog(text string, conn *net.UDPConn) {
	logCache = append(logCache, text)
	leng := len(logCache)
	if leng >= 5 {
		conn.Write([]byte(strings.Join(logCache, "")))
	}
	if time.Now().After(begin.Add(maxSendTime)) && leng > 0 {
		conn.Write([]byte(strings.Join(logCache, "")))
	}
	logCache = make([]string, 5)
	begin = time.Now()
}
