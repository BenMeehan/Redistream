package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

func readCommand(reader *bufio.Reader) ([]string, error) {
	// Read the first line, which contains the command array length
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	// Parse the command array length
	length, err := strconv.Atoi(strings.TrimSpace(line[1:]))
	if err != nil {
		return nil, err
	}

	// Initialize slice to store command elements
	commands := make([]string, 0)

	// Read each command element
	for i := 0; i < length; i++ {
		// Read the line containing the length of the command element
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		// Parse the length of the command element
		elementLength, err := strconv.Atoi(strings.TrimSpace(line[1:]))
		if err != nil {
			return nil, err
		}

		// Read the command element
		element := make([]byte, elementLength)
		_, err = reader.Read(element)
		if err != nil {
			return nil, err
		}

		// Append the command element to the commands slice
		commands = append(commands, string(element))

		// Read the trailing '\r\n'
		_, err = reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
	}

	return commands, nil
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Println("Client connected from", conn.RemoteAddr())

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		// Read the command from the client
		commands, err := readCommand(reader)
		if err != nil {
			fmt.Println("Error reading command:", err)
			return
		}

		for _, cmd := range commands {
			if strings.ToUpper(cmd) == "PING" {
				// Respond with +PONG
				response := "+PONG\r\n"
				writer.WriteString(response)
			} else {
				// Respond with an error for unsupported commands
				response := "-ERR unknown command\r\n"
				writer.WriteString(response)
			}

			err = writer.Flush()
			if err != nil {
				fmt.Println("Error writing response:", err)
				return
			}
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
