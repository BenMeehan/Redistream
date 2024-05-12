package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type serverConfig struct {
	port       int
	role       string
	replid     string
	replOffset int
	masterHost string
	masterPort int
}

type serverState struct {
	store         map[string]string
	ttl           map[string]time.Time
	config        serverConfig
	replicas      []replica
	replicaOffset int
}

func main() {

	var config serverConfig

	flag.IntVar(&config.port, "port", 6379, "listen on specified port")
	flag.StringVar(&config.masterHost, "replicaof", "", "start server in replica mode of given host and port")
	flag.Parse()

	if len(config.masterHost) == 0 {
		config.role = "master"
		config.replid = randReplid()
	} else {
		config.role = "slave"
		switch flag.NArg() {
		case 0:
			config.masterPort = 6379
		case 1:
			config.masterPort, _ = strconv.Atoi(flag.Arg(0))
		default:
			flag.Usage()
		}
	}

	srv := newServer(config)

	srv.start()
}

func newServer(config serverConfig) *serverState {
	var srv serverState
	srv.store = make(map[string]string)
	srv.ttl = make(map[string]time.Time)
	srv.config = config
	return &srv
}

func (srv *serverState) start() {
	if srv.config.role == "slave" {
		srv.replicaHandshake()
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", srv.config.port))
	if err != nil {
		fmt.Printf("Failed to bind to port %d\n", srv.config.port)
		os.Exit(1)
	}
	fmt.Println("Listening on: ", listener.Addr().String())

	for id := 1; ; id++ {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		go srv.serveClient(id, conn)
	}
}

func (srv *serverState) serveClient(id int, conn net.Conn) {
	fmt.Printf("[#%d] Client connected: %v\n", id, conn.RemoteAddr().String())

	reader := bufio.NewReader(conn)

	for {
		cmd, _, err := decodeStringArray(reader)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("[%d] Error decoding command: %v\n", id, err.Error())
		}

		if len(cmd) == 0 {
			break
		}

		fmt.Printf("[#%d] Command = %q\n", id, cmd)
		response, resynch := srv.handleCommand(cmd)

		if len(response) > 0 {
			bytesSent, err := conn.Write([]byte(response))
			if err != nil {
				fmt.Printf("[#%d] Error writing response: %v\n", id, err.Error())
				break
			}
			fmt.Printf("[#%d] Bytes sent: %d %q\n", id, bytesSent, response)
		}

		if resynch {
			size := sendFullResynch(conn)
			fmt.Printf("[#%d] full resynch sent: %d\n", id, size)
			srv.replicas = append(srv.replicas, replica{conn, 0})
			fmt.Printf("[#%d] Client promoted to replica\n", id)
			return
		}
	}

	fmt.Printf("[#%d] Client closing\n", id)
	conn.Close()
}

func (srv *serverState) handleCommand(cmd []string) (response string, resynch bool) {

	switch strings.ToUpper(cmd[0]) {
	case "PING":
		response = "+PONG\r\n"

	case "ECHO":
		response = encodeBulkString(cmd[1])

	case "INFO":
		if len(cmd) == 2 && strings.ToUpper(cmd[1]) == "REPLICATION" {
			response = encodeBulkString(fmt.Sprintf("role:%s\r\nmaster_replid:%s\r\nmaster_repl_offset:%d",
				srv.config.role, srv.config.replid, srv.config.replOffset))
		}

	case "SET":
		key, value := cmd[1], cmd[2]
		srv.store[key] = value
		if len(cmd) == 5 && strings.ToUpper(cmd[3]) == "PX" {
			expiration, _ := strconv.Atoi(cmd[4])
			srv.ttl[key] = time.Now().Add(time.Millisecond * time.Duration(expiration))
		}
		response = "+OK\r\n"
		srv.propagateToReplicas(cmd)

	case "GET":
		key := cmd[1]
		value, ok := srv.store[key]
		if ok {
			expiration, exists := srv.ttl[key]
			if !exists || expiration.After(time.Now()) {
				response = encodeBulkString(value)
			} else if exists {
				delete(srv.ttl, key)
				delete(srv.store, key)
				response = encodeBulkString("")
			}
		} else {
			response = encodeBulkString("")
		}

	case "REPLCONF":
		switch strings.ToUpper(cmd[1]) {
		case "GETACK":
			response = encodeStringArray([]string{"REPLCONF", "ACK", fmt.Sprint(srv.replicaOffset)})
		default:
			response = "+OK\r\n"
		}

	case "PSYNC":
		if len(cmd) == 3 {
			response = fmt.Sprintf("+FULLRESYNC %s 0\r\n", srv.config.replid)
			resynch = true
		}

	case "WAIT":
		if len(cmd) == 3 {
			response = encodeInteger(0)
		}

	}

	return
}
