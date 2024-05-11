package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// In-memory map to store the redis key-value pairs
var KeyValuePairs = make(map[string]string)

// Map to store expiry time for each key
var KeyExpiryTime = make(map[string]int64)

var port = 6379
var isReplica bool
var masterReplID string
var masterReplOffset int
var masterHost string
var masterPort int

var replicas = make([]net.Conn, 0)

// handleConnection handles commands from a client connection.
func handleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Println("Client connected from", conn.RemoteAddr())

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		commands, err := ReadCommand(reader)
		if err != nil {
			fmt.Println("Error reading command:", err)
			return
		}

		fmt.Println(commands)
		for i := 0; i < len(commands); i++ {
			cmd := commands[i]
			var response string
			var file []byte
			switch strings.ToUpper(cmd) {
			case "PING":
				response = Ping()
			case "ECHO":
				response, i = Echo(i, commands)
			case "SET":
				response, i = Set(i, commands)
				PropagateToReplicas(replicas, commands)
			case "GET":
				response, i = Get(i, commands)
			case "INFO":
				response, i = Info(i, commands)
			case "REPLCONF":
				response, i = HandleREPLCONF(i, commands)
			case "PSYNC":
				response, i = Psync(i)
				file = SendEmptyRDBFile(conn)
				replicas = append(replicas, conn)
			default:
				response = "-ERR unknown command\r\n"
			}

			if cmd == "SET" && isReplica {
				continue
			}

			err := WriteResponse(writer, response)
			if err != nil {
				fmt.Println("Error writing response:", err)
				return
			}

			if len(file) > 0 {
				err := WriteResponse(writer, string(file))
				if err != nil {
					fmt.Println("Error writing file:", err)
					return
				}
			}
		}
	}
}

func main() {
	var err error
	i := 1
	args := os.Args
	fmt.Println(args)
	for i < len(args) {
		switch args[i] {
		case "--port":
			port, err = strconv.Atoi(args[i+1])
			if err != nil {
				fmt.Println("Invalid port")
				os.Exit(1)
			}
			i += 2
		case "--replicaof":
			isReplica = true
			masterHost = args[i+1]
			if len(masterHost) == 0 {
				fmt.Println("Invalid master hostname")
				os.Exit(1)
			}
			masterPort, err = strconv.Atoi(args[i+2])
			if err != nil {
				fmt.Println("Invalid master port")
				os.Exit(1)
			}
			i += 3
		default:
			i++
		}
	}

	masterReplID = "8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb"
	masterReplOffset = 0

	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		fmt.Println("Failed to bind to port", port, ":", err)
		os.Exit(1)
	}
	defer l.Close()

	fmt.Println("Server listening on port", port)

	if isReplica {
		go ConnectToMasterHandshake(masterHost, masterPort)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn) // Handle each client connection concurrently
	}
}
