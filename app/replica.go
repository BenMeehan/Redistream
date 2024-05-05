package main

import (
	"fmt"
	"net"
)

// ConnectToMaster establishes a connection from the replica to the master.
func ConnectToMaster(masterHost string, masterPort int) {
	fmt.Printf("Connecting to master %s:%d\n", masterHost, masterPort)
	masterAddr := fmt.Sprintf("%s:%d", masterHost, masterPort)
	conn, err := net.Dial("tcp", masterAddr)
	if err != nil {
		fmt.Println("Error connecting to master:", err)
		return
	}
	defer conn.Close()

	// Send the PING command to the master
	pingCommand := "*1\r\n$4\r\nPING\r\n"
	_, err = conn.Write([]byte(pingCommand))
	if err != nil {
		fmt.Println("Error sending PING command to master:", err)
		return
	}
	fmt.Println("Sent PING command to master")
}
