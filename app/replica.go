package main

import (
	"fmt"
	"net"
	"strconv"
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

	// Send the REPLCONF command with listening-port
	listeningPortCommand := "*3\r\n$8\r\nREPLCONF\r\n$14\r\nlistening-port\r\n$" + strconv.Itoa(len(strconv.Itoa(port))) + "\r\n" + strconv.Itoa(port) + "\r\n"
	_, err = conn.Write([]byte(listeningPortCommand))
	if err != nil {
		fmt.Println("Error sending REPLCONF listening-port command to master:", err)
		return
	}
	fmt.Println("Sent REPLCONF listening-port command to master")

	// Send the REPLCONF command with capa psync2
	capaCommand := "*3\r\n$8\r\nREPLCONF\r\n$4\r\ncapa\r\n$6\r\npsync2\r\n"
	_, err = conn.Write([]byte(capaCommand))
	if err != nil {
		fmt.Println("Error sending REPLCONF capa command to master:", err)
		return
	}
	fmt.Println("Sent REPLCONF capa command to master")
}
