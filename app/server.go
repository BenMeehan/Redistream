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

var isReplica bool
var masterReplID string
var masterReplOffset int
var masterHost string
var masterPort int

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

		for i := 0; i < len(commands); i++ {
			cmd := commands[i]
			var response string
			switch strings.ToUpper(cmd) {
			case "PING":
				response = Ping()
			case "ECHO":
				response = Echo(i, commands)
			case "SET":
				response = Set(i, commands)
			case "GET":
				response = Get(i, commands)
			case "INFO":
				response = Info(i, commands)
			default:
				response = "-ERR unknown command\r\n"
			}

			err := WriteResponse(writer, response)
			if err != nil {
				fmt.Println("Error writing response:", err)
				return
			}
		}
	}
}

func main() {
	var err error
	i := 1
	port := 6379
	args := os.Args
	fmt.Println(args)
	for i < len(args) {
		switch args[i] {
		case "--port":
			port, err = strconv.Atoi(args[i+2])
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

	fmt.Println(masterHost, masterPort)

	masterReplID = "8371b4fb1155b71f4a04d3e1bc3e18c4a990aeeb"
	masterReplOffset = 0

	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		fmt.Println("Failed to bind to port", port, ":", err)
		os.Exit(1)
	}
	defer l.Close()

	fmt.Println("Server listening on port", port)

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn) // Handle each client connection concurrently
	}
}
