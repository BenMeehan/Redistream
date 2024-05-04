package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

func readCommand(reader *bufio.Reader) (string, error) {
	// Read the command length
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	length, err := strconv.Atoi(strings.TrimSpace(line[1:]))
	if err != nil {
		return "", err
	}

	// Read the command
	command := make([]byte, length)
	_, err = reader.Read(command)
	if err != nil {
		return "", err
	}
	// Read the trailing '\r\n'
	reader.ReadByte()
	reader.ReadByte()

	return string(command), nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Println("Client connected from", conn.RemoteAddr())

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		// Read the command from the client
		command, err := readCommand(reader)
		if err != nil {
			fmt.Println("Error reading command:", err)
			return
		}

		// Check the command type
		if strings.ToUpper(command) == "PING" {
			// Respond with +PONG
			response := "+PONG\r\n"
			writer.WriteString(response)
		} else {
			// Respond with an error for unsupported commands
			response := "-ERR unknown command\r\n"
			writer.WriteString(response)
		}

		// Flush the writer to send the response immediately
		err = writer.Flush()
		if err != nil {
			fmt.Println("Error writing response:", err)
			return
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
