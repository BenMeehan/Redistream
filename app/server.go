package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

// readCommand reads and parses a Redis command from the client connection.
func readCommand(reader *bufio.Reader) ([]string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	length, err := strconv.Atoi(strings.TrimSpace(line[1:]))
	if err != nil {
		return nil, err
	}

	commands := make([]string, 0)

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

// writeResponse writes a response to the client connection.
func writeResponse(writer *bufio.Writer, response string) error {
	_, err := writer.WriteString(response)
	if err != nil {
		return err
	}
	return writer.Flush()
}

// handleConnection handles commands from a client connection.
func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		commands, err := readCommand(reader)
		if err != nil {
			fmt.Println("Error reading command:", err)
			return
		}

		for i := 0; i < len(commands); i++ {
			cmd := commands[i]
			var response string
			switch strings.ToUpper(cmd) {
			case "PING":
				response = "+PONG\r\n"
			case "ECHO":
				if i < len(commands)-1 {
					response = fmt.Sprintf("$%d\r\n%s\r\n", len(commands[i+1]), commands[i+1])
					i++
				} else {
					response = "-ERR wrong number of arguments for 'echo' command\r\n"
				}
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
			continue
		}
		go handleConnection(conn) // Handle each client connection concurrently
	}
}
