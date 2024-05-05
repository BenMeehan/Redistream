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
	command := "*1\r\n$4\r\nPING\r\n"
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
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading response:", err)
		return
	}

	// Print the response
	fmt.Println("Response from server:", response)

	// You can add more commands here as needed
}
