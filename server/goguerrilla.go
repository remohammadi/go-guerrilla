/**
Go-Guerrilla SMTPd

Version: 1.5
Author: Flashmob, GuerrillaMail.com
Contact: flashmob@gmail.com
License: MIT
Repository: https://github.com/flashmob/Go-Guerrilla-SMTPd
Site: http://www.guerrillamail.com/

See README for more details
*/

package server

import (
	"bufio"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net"
	"runtime"
	"strconv"
	"time"
)

var allowedHosts = make(map[string]bool, 15)

// // map the allow hosts for easy lookup
// if len(mainConfig.Allowed_hosts) > 0 {
// 	if arr := strings.Split(mainConfig.Allowed_hosts, ","); len(arr) > 0 {
// 		for i := 0; i < len(arr); i++ {
// 			allowedHosts[arr[i]] = true
// 		}
// 	}
// } else {
// 	log.Fatalln("Config error, GM_ALLOWED_HOSTS must be s string.")
// }

func RunServer(sConfig ServerConfig, backend guerrilla.Backend) (err error) {
	server := SmtpdServer{
		Config: sConfig,
		sem: make(chan int, sConfig.Max_clients)
	}

	// setup logging
	server.openLog()

	// configure ssl
	if sConfig.Tls_always_on || sConfig.Start_tls_on {
		cert, err := tls.LoadX509KeyPair(sConfig.Public_key_file, sConfig.Private_key_file)
		if err != nil {
			server.logln(2, fmt.Sprintf("There was a problem with loading the certificate: %s", err))
		}
		server.tlsConfig = &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.VerifyClientCertIfGiven,
			ServerName:   sConfig.Host_name,
		}
		server.tlsConfig.Rand = rand.Reader
	}

	// configure timeout
	server.timeout = time.Duration(sConfig.Timeout)

	// Start listening for SMTP connections
	listener, err := net.Listen("tcp", sConfig.Listen_interface)
	if err != nil {
		server.logln(2, fmt.Sprintf("Cannot listen on port, %v", err))
		return err
	} else {
		server.logln(1, fmt.Sprintf("Listening on tcp %s", sConfig.Listen_interface))
	}
	var clientId int64
	clientId = 1
	for {
		conn, err := listener.Accept()
		if err != nil {
			server.logln(1, fmt.Sprintf("Accept error: %s", err))
			continue
		}
		server.logln(0, fmt.Sprintf(" There are now "+strconv.Itoa(runtime.NumGoroutine())+" serving goroutines"))
		server.sem <- 1 // Wait for active queue to drain.
		go server.handleClient(&Client{
			conn:        conn,
			address:     conn.RemoteAddr().String(),
			time:        time.Now().Unix(),
			bufin:       newSmtpBufferedReader(conn),
			bufout:      bufio.NewWriter(conn),
			clientId:    clientId,
			savedNotify: make(chan int),
		})
		clientId++
	}
}
