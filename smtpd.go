package guerrilla

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const commandMaxLength = 1024

type Client struct {
	state       int
	helo        string
	mail_from   string
	rcpt_to     string
	response    string
	address     string
	data        string
	subject     string
	hash        string
	time        int64
	tls_on      bool
	conn        net.Conn
	bufin       *smtpBufferedReader
	bufout      *bufio.Writer
	kill_time   int64
	errors      int
	clientId    int64
	savedNotify chan int
}

type SmtpdServer struct {
	tlsConfig    *tls.Config
	max_size     int // max email DATA size
	timeout      time.Duration
	allowedHosts map[string]bool
	sem          chan int // currently active client list
	Config       ServerConfig
	logger       *log.Logger
}

func (server *SmtpdServer) logln(level int, s string) {

	if mainConfig.Verbose {
		fmt.Println(s)
	}
	// fatal errors
	if level == 2 {
		server.logger.Fatalf(s)
	}
	// warnings
	if level == 1 && len(server.Config.Log_file) > 0 {
		server.logger.Println(s)
	}

}

func (server *SmtpdServer) openLog() {

	server.logger = log.New(&bytes.Buffer{}, "", log.Lshortfile)
	// custom log file
	if len(server.Config.Log_file) > 0 {
		logfile, err := os.OpenFile(
			server.Config.Log_file,
			os.O_WRONLY|os.O_APPEND|os.O_CREATE|os.O_SYNC, 0600)
		if err != nil {
			server.logln(1, fmt.Sprintf("Unable to open log file [%s]: %s ", server.Config.Log_file, err))
		}
		server.logger.SetOutput(logfile)
	}
}

// Upgrades the connection to TLS
// Sets up buffers with the upgraded connection
func (server *SmtpdServer) upgradeToTls(client *Client) bool {
	var tlsConn *tls.Conn
	tlsConn = tls.Server(client.conn, server.tlsConfig)
	err := tlsConn.Handshake()
	if err == nil {
		client.conn = net.Conn(tlsConn)
		client.bufin = newSmtpBufferedReader(client.conn)
		client.bufout = bufio.NewWriter(client.conn)
		client.tls_on = true

		return true
	} else {
		server.logln(1, fmt.Sprintf("Could not TLS handshake:%v", err))
		return false
	}

}

func (server *SmtpdServer) handleClient(client *Client) {
	defer server.closeClient(client)
	advertiseTls := "250-STARTTLS\r\n"
	if server.Config.Tls_always_on {
		if server.upgradeToTls(client) {
			advertiseTls = ""
		}
	}
	greeting := "220 " + server.Config.Host_name +
		" SMTP Guerrilla-SMTPd #" +
		strconv.FormatInt(client.clientId, 10) +
		" (" + strconv.Itoa(len(server.sem)) + ") " + time.Now().Format(time.RFC1123Z)

	if !server.Config.Start_tls_on {
		// STARTTLS turned off
		advertiseTls = ""
	}
	for i := 0; i < 100; i++ {
		switch client.state {
		case 0:
			responseAdd(client, greeting)
			client.state = 1
		case 1:
			client.bufin.setLimit(commandMaxLength)
			input, err := server.readSmtp(client)
			if err != nil {
				if err == io.EOF {
					// client closed the connection already
					server.logln(0, fmt.Sprintf("%s: %v", client.address, err))
					return
				} else if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
					// too slow, timeout
					server.logln(0, fmt.Sprintf("%s: %v", client.address, err))
					return
				} else if err == INPUT_LIMIT_EXCEEDED {
					responseAdd(client, "500 Line too long.")
					// kill it so that another one can connect
					killClient(client)
				}
				server.logln(1, fmt.Sprintf("Read error: %v", err))
				break
			}
			input = strings.Trim(input, " \n\r")
			bound := len(input)
			if bound > 16 {
				bound = 16
			}
			cmd := strings.ToUpper(input[0:bound])
			switch {
			case strings.Index(cmd, "HELO") == 0:
				if len(input) > 5 {
					client.helo = input[5:]
				}
				responseAdd(client, "250 "+server.Config.Host_name+" Hello ")
			case strings.Index(cmd, "EHLO") == 0:
				if len(input) > 5 {
					client.helo = input[5:]
				}
				responseAdd(client, "250-"+server.Config.Host_name+
					" Hello "+client.helo+"["+client.address+"]"+"\r\n"+
					"250-SIZE "+strconv.Itoa(server.Config.Max_size)+"\r\n"+
					"250-PIPELINING \r\n"+
					advertiseTls+"250 HELP")
			case strings.Index(cmd, "HELP") == 0:
				responseAdd(client, "250 Help! I need somebody...")
			case strings.Index(cmd, "MAIL FROM:") == 0:
				if len(input) > 10 {
					client.mail_from = input[10:]
				}
				responseAdd(client, "250 Ok")
			case strings.Index(cmd, "XCLIENT") == 0:
				// Nginx sends this
				// XCLIENT ADDR=212.96.64.216 NAME=[UNAVAILABLE]
				client.address = input[13:]
				client.address = client.address[0:strings.Index(client.address, " ")]
				fmt.Println("client address:[" + client.address + "]")
				responseAdd(client, "250 OK")
			case strings.Index(cmd, "RCPT TO:") == 0:
				if len(input) > 8 {
					client.rcpt_to = input[8:]
				}
				responseAdd(client, "250 Accepted")
			case strings.Index(cmd, "NOOP") == 0:
				responseAdd(client, "250 OK")
			case strings.Index(cmd, "RSET") == 0:
				client.mail_from = ""
				client.rcpt_to = ""
				responseAdd(client, "250 OK")
			case strings.Index(cmd, "DATA") == 0:
				responseAdd(client, "354 Enter message, ending with \".\" on a line by itself")
				client.state = 2
			case (strings.Index(cmd, "STARTTLS") == 0) &&
				!client.tls_on &&
				server.Config.Start_tls_on:
				responseAdd(client, "220 Ready to start TLS")
				// go to start TLS state
				client.state = 3
			case strings.Index(cmd, "QUIT") == 0:
				responseAdd(client, "221 Bye")
				killClient(client)
			default:
				responseAdd(client, "500 unrecognized command: "+cmd)
				client.errors++
				if client.errors > 3 {
					responseAdd(client, "500 Too many unrecognized commands")
					killClient(client)
				}
			}
		case 2:
			var err error
			client.bufin.setLimit(int64(server.Config.Max_size) + 1024000) // This is a hard limit.
			client.data, err = server.readSmtp(client)
			if err == nil {
				if _, _, mailErr := validateEmailData(client); mailErr == nil {
					// to do: timeout when adding to SaveMailChan
					// place on the channel so that one of the save mail workers can pick it up
					SaveMailChan <- &savePayload{client: client, server: server}
					// wait for the save to complete
					// or timeout
					select {
					case status := <-client.savedNotify:
						if status == 1 {
							responseAdd(client, "250 OK : queued as "+client.hash)
						} else {
							responseAdd(client, "554 Error: transaction failed, blame it on the weather")
						}
					case <-time.After(time.Second * 30):
						fmt.Println("timeout 1")
						responseAdd(client, "554 Error: transaction timeout")
					}

				} else {
					responseAdd(client, "550 Error: "+mailErr.Error())
				}

			} else {
				if (err == INPUT_LIMIT_EXCEEDED) {
					// hard limit reached, end to make room for other clients
					responseAdd(client, "550 Error: DATA limit exceeded by more than a megabyte!")
					killClient(client)
				} else {
					responseAdd(client, "550 Error: "+err.Error())
				}

				server.logln(1, fmt.Sprintf("DATA read error: %v", err))
			}
			client.state = 1
		case 3:
			// upgrade to TLS
			if server.upgradeToTls(client) {
				advertiseTls = ""
				client.state = 1
			}
		}
		// Send a response back to the client
		err := server.responseWrite(client)
		if err != nil {
			if err == io.EOF {
				// client closed the connection already
				return
			}
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				// too slow, timeout
				return
			}
		}
		if client.kill_time > 1 {
			return
		}
	}

}

// add a response on the response buffer
func responseAdd(client *Client, line string) {
	client.response = line + "\r\n"
}
func (server *SmtpdServer) closeClient(client *Client) {
	client.conn.Close()
	<-server.sem // Done; enable next client to run.
}
func killClient(client *Client) {
	client.kill_time = time.Now().Unix()
}

var INPUT_LIMIT_EXCEEDED = errors.New("Line too long") // 500 Line too long.

// we need to adjust the limit, so we embed io.LimitedReader
type adjustableLimitedReader struct {
	R *io.LimitedReader
}

// bolt this on so we can adjust the limit
func (alr *adjustableLimitedReader) setLimit(n int64) {
	alr.R.N = n
}

// this just delegates to the underlying reader in order to satisfy the Reader interface
// Since the vanilla limited reader returns io.EOF when the limit is reached, we need a more specific
// error so that we can distinguish when a limit is reached
func (alr *adjustableLimitedReader) Read(p []byte) (n int, err error) {
	n, err = alr.R.Read(p)
	if err == io.EOF && alr.R.N <= 0 {
		// return our custom error since std lib returns EOF
		err = INPUT_LIMIT_EXCEEDED
	}
	return
}

// allocate a new adjustableLimitedReader
func newAdjustableLimitedReader(r io.Reader, n int64) *adjustableLimitedReader {
	lr := &io.LimitedReader{R: r, N: n}
	return &adjustableLimitedReader{lr}
}

// This is a bufio.Reader what will use our adjustable limit reader
// We 'extend' buffio to have the limited reader feature
type smtpBufferedReader struct {
	*bufio.Reader
	alr *adjustableLimitedReader
}


// delegate to the adjustable limited reader
func (sbr *smtpBufferedReader) setLimit(n int64) {
	sbr.alr.setLimit(n)
}


// allocate a new smtpBufferedReader
func newSmtpBufferedReader(rd io.Reader) *smtpBufferedReader {
	alr := newAdjustableLimitedReader(rd, commandMaxLength)
	s := &smtpBufferedReader{bufio.NewReader(alr), alr}
	return s
}

// Reads from the smtpBufferedReader, can be in command state or data state.
func (server *SmtpdServer) readSmtp(client *Client) (input string, err error) {
	var reply string
	// Command state terminator by default
	suffix := "\r\n"
	if client.state == 2 {
		// DATA state ends with a dot on a line by itself
		suffix = "\r\n.\r\n"
	}
	for err == nil {
		client.conn.SetDeadline(time.Now().Add(server.timeout * time.Second))
		reply, err = client.bufin.ReadString('\n')
		if reply != "" {
			input = input + reply
			if len(input) > server.Config.Max_size {
				err = errors.New("Maximum DATA size exceeded (" + strconv.Itoa(server.Config.Max_size) + ")")
				return input, err
			}
			if client.state == 2 {
				// Extract the subject while we are at it.
				scanSubject(client, reply)
			}
		}
		if err != nil {
			break
		}
		if strings.HasSuffix(input, suffix) {
			break
		}
	}
	return input, err
}

// Scan the data part for a Subject line. Can be a multi-line
func scanSubject(client *Client, reply string) {
	if client.subject == "" && (len(reply) > 8) {
		test := strings.ToUpper(reply[0:9])
		if i := strings.Index(test, "SUBJECT: "); i == 0 {
			// first line with \r\n
			client.subject = reply[9:]
		}
	} else if strings.HasSuffix(client.subject, "\r\n") {
		// chop off the \r\n
		client.subject = client.subject[0 : len(client.subject)-2]
		if (strings.HasPrefix(reply, " ")) || (strings.HasPrefix(reply, "\t")) {
			// subject is multi-line
			client.subject = client.subject + reply[1:]
		}
	}
}

func (server *SmtpdServer) responseWrite(client *Client) (err error) {
	var size int
	client.conn.SetDeadline(time.Now().Add(server.timeout * time.Second))
	size, err = client.bufout.WriteString(client.response)
	client.bufout.Flush()
	client.response = client.response[size:]
	return err
}
