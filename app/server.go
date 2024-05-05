package main

import (
	"bufio"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// In-memory map to store the redis key-value pairs
var KeyValuePairs = make(map[string]string)

// Map to store expiry time for each key
var KeyExpiryTime = make(map[string]int64)

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

	fmt.Println("Client connected from", conn.RemoteAddr())

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
			case "SET":
				if i < len(commands)-2 {
					key := commands[i+1]
					value := commands[i+2]
					KeyValuePairs[key] = value
					fmt.Printf("SET %s %s\n", key, value)

					if i+3 < len(commands) && strings.ToUpper(commands[i+3]) == "PX" {
						expiry, err := strconv.Atoi(commands[i+4])
						if err != nil {
							response = "-ERR invalid expiry\r\n"
							break
						}

						KeyExpiryTime[key] = time.Now().UnixNano() + int64(expiry)*int64(time.Millisecond)

						/* Active Expiry: Working Idea
						go func(k string, exp int) {
							time.Sleep(time.Duration(exp) * time.Millisecond)
							delete(KeyValuePairs, k)
							fmt.Printf("Key %s expired\n", k)
						}(key, expiry)
						*/
						i += 4
					} else {
						i += 2
					}
					response = "+OK\r\n"
				} else {
					response = "-ERR wrong number of arguments for 'set' command\r\n"
				}
			case "GET":
				if i < len(commands)-1 {
					key := commands[i+1]
					fmt.Printf("GET %s\n", key)
					if value, ok := KeyValuePairs[key]; ok {
						if expiry, found := KeyExpiryTime[key]; found && expiry <= time.Now().UnixNano() {
							// Key has expired, delete it
							delete(KeyValuePairs, key)
							delete(KeyExpiryTime, key)
							response = fmt.Sprintf("$%d\r\n", -1)
						} else {
							response = fmt.Sprintf("$%d\r\n%s\r\n", len(value), value)
						}
						i++
					} else {
						response = fmt.Sprintf("$%d\r\n", -1)
						i++
					}
				} else {
					response = "-ERR wrong number of arguments for 'get' command\r\n"
				}
			case "INFO":
				if i < len(commands)-1 && strings.ToUpper(commands[i+1]) == "REPLICATION" {
					response = "$13\r\n# Replication\r\n$15\r\nrole:master\r\n"
					i++
				} else {
					response = "$13\r\n# Replication\r\n"
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
	port := flag.Int("port", 6379, "Port number for the Redis server")
	flag.Parse()

	l, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *port))
	if err != nil {
		fmt.Println("Failed to bind to port", *port, ":", err)
		os.Exit(1)
	}
	defer l.Close()

	fmt.Println("Server listening on port", *port)

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go handleConnection(conn) // Handle each client connection concurrently
	}
}
