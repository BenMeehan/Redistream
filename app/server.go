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
	// Read the command array length
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	length, err := strconv.Atoi(strings.TrimSpace(line[1:]))
	if err != nil {
		return nil, err
	}

	// Read each command element
	var commands []string
	for i := 0; i < length; i++ {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		elementLength, err := strconv.Atoi(strings.TrimSpace(line[1:]))
		if err != nil {
			return nil, err
		}
		element := make([]byte, elementLength)
		_, err = reader.Read(element)
		if err != nil {
			return nil, err
		}
		commands = append(commands, string(element))
		_, err = reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
	}

	return commands, nil
}

func writeResponse(writer *bufio.Writer, response string) error {
	_, err := writer.WriteString(response)
	if err != nil {
		return err
	}
	return writer.Flush()
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Println("Client connected from", conn.RemoteAddr())

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		commands, err := readCommand(reader)
		if err != nil {
			fmt.Println("Error reading command:", err)
			return
		}

		for _, cmd := range commands {
			var response string
			switch strings.ToUpper(cmd) {
			case "PING":
				response = "+PONG\r\n"
			default:
				response = "-ERR unknown command\r\n"
			}

			err := writeResponse(writer, response)
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
		fmt.Println("Failed to bind to port 6379:", err)
		os.Exit(1)
	}
	defer l.Close()

	fmt.Println("Server listening on port 6379")

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			os.Exit(1)
		}
		go handleConnection(conn)
	}
}
