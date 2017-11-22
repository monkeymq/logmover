// Copyright (c) 2017 The logmover Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	t, err := tail.TailFile(logfile, tail.Config{Follow: true, ReOpen: true})
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
