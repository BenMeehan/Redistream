package main

import (
	"bufio"
	"fmt"
	"net"
)

func main() {
	// Connect to the Redis server
	conn, err := net.Dial("tcp", "localhost:6379")
	if err != nil {
		fmt.Println("Error connecting to server:", err)
		return
	}
	defer conn.Close()

	// Create a bufio reader and writer for reading/writing data from/to the server
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Send a PING command to the server
	command := "*3\r\n$4\r\nWAIT\r\n$1\r\n2\r\n$4\r\n1000\r\n"
	_, err = writer.WriteString(command)
	if err != nil {
		fmt.Println("Error writing command:", err)
		return
	}
	err = writer.Flush()
	if err != nil {
		fmt.Println("Error flushing writer:", err)
		return
	}

	// Read the response from the server
	var response string
	for {
		line, err := reader.ReadString('\n')
		fmt.Println(line)
		if err != nil {
			fmt.Println("Error reading response:", err)
			return
		}
		response += line
		if line == "\r\n" { // Check for end of response
			break
		}
	}

	// Print the response
	fmt.Println("Response from server:", response)

}
