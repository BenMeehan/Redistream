package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Println("Client connected from", conn.RemoteAddr())

	// Create a bufio reader to read input from the connection
	reader := bufio.NewReader(conn)

	for {
		// Read the command from the client
		command, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading command:", err)
			return
		}

		// Check if the command is PING
		if command == "PING\r\n" {
			// Respond with +PONG
			response := "+PONG\r\n"
			conn.Write([]byte(response))
		} else {
			// Respond with an error for unsupported commands
			response := "-ERR unknown command\r\n"
			conn.Write([]byte(response))
		}
	}
}

func main() {
	l, err := net.Listen("tcp", "0.0.0.0:6379")
	if err != nil {
		fmt.Println("Failed to bind to port 6379")
		os.Exit(1)
	}

	defer l.Close()

	fmt.Println("Server listening on port 6379")

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		go handleConnection(conn)
	}
}
