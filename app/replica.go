package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

var replicaPort = 6379
var replicationId = "?"
var replicationOffset = -1

// ConnectToMaster establishes a connection from the replica to the master.
func ConnectToMasterHandshake(masterHost string, masterPort int) {
	fmt.Printf("Connecting to master %s:%d\n", masterHost, masterPort)
	masterAddr := fmt.Sprintf("%s:%d", masterHost, masterPort)
	conn, err := net.Dial("tcp", masterAddr)
	if err != nil {
		fmt.Println("Error connecting to master:", err)
		return
	}

	reader := bufio.NewReader(conn)

	// Send the PING command to the master
	pingCommand := "*1\r\n$4\r\nPING\r\n"
	_, err = conn.Write([]byte(pingCommand))
	if err != nil {
		fmt.Println("Error sending PING command to master:", err)
		return
	}
	fmt.Println("Sent PING command to master")

	// Read the response from the master
	reader.ReadString('\n')

	// Send the REPLCONF command with listening-port
	listeningPortCommand := "*3\r\n$8\r\nREPLCONF\r\n$14\r\nlistening-port\r\n$" + strconv.Itoa(len(strconv.Itoa(port))) + "\r\n" + strconv.Itoa(port) + "\r\n"
	_, err = conn.Write([]byte(listeningPortCommand))
	if err != nil {
		fmt.Println("Error sending REPLCONF listening-port command to master:", err)
		return
	}
	fmt.Println("Sent REPLCONF listening-port command to master")

	// Read the response from the master
	reader.ReadString('\n')

	// Send the REPLCONF command with capa psync2
	capaCommand := "*3\r\n$8\r\nREPLCONF\r\n$4\r\ncapa\r\n$6\r\npsync2\r\n"
	_, err = conn.Write([]byte(capaCommand))
	if err != nil {
		fmt.Println("Error sending REPLCONF capa command to master:", err)
		return
	}
	fmt.Println("Sent REPLCONF capa command to master")

	// Read the response from the master
	reader.ReadString('\n')

	// Send the PSYNC command
	psyncCommand := fmt.Sprintf("*3\r\n$5\r\nPSYNC\r\n$1\r\n%s\r\n$2\r\n%d\r\n", replicationId, replicationOffset)
	_, err = conn.Write([]byte(psyncCommand))
	if err != nil {
		fmt.Println("Error sending PSYNC command to master:", err)
		return
	}
	fmt.Println("Sent PSYNC command to master")

	// Read the response from the master
	reader.ReadString('\n')

	// v := strings.Split(string(response[:n]), " ")
	// replicationId = v[1]
	// replicationOffset, err = strconv.Atoi(v[2])
	// if err != nil {
	// 	fmt.Println("Invalid replication offset from master:", err)
	// 	return
	// }

	// fmt.Println("Updated master replication id and offset:", replicationId, replicationOffset)
	// Read the response from the master
	// receiving RDB (ignoring it for now)
	response, _ := reader.ReadString('\n')
	if response[0] != '$' {
		fmt.Printf("Invalid response\n")
		os.Exit(1)
	}
	rdbSize, _ := strconv.Atoi(response[1 : len(response)-2])
	buffer := make([]byte, rdbSize)
	receivedSize, err := reader.Read(buffer)
	if err != nil {
		fmt.Printf("Invalid RDB received %v\n", err)
		os.Exit(1)
	}
	if rdbSize != receivedSize {
		fmt.Printf("Size mismatch - got: %d, want: %d\n", receivedSize, rdbSize)
	}

	go handlePropagation(conn)
}

// ReceiveRDBFile receives the RDB file from the master.
func ReceiveRDBFile(conn net.Conn) error {
	fmt.Println("Receiving RDB file from master")

	reader := bufio.NewReader(conn)

	// Read the length of the RDB file
	lengthStr, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	lengthStr = strings.TrimSpace(lengthStr)
	length, err := strconv.Atoi(lengthStr[1:])
	if err != nil {
		return err
	}

	// Read the RDB file contents
	rdbData := make([]byte, length)
	_, err = io.ReadFull(reader, rdbData)
	if err != nil {
		return err
	}

	fmt.Println("Received RDB file from master", string(rdbData))

	return nil
}

func handlePropagation(conn net.Conn) {
	fmt.Println("handling propagation")
	reader := bufio.NewReader(conn)
	for {
		commands, err := ReadCommand(reader)
		if err != nil {
			fmt.Println("Error reading command:", err)
			return
		}
		for i := 0; i < len(commands); i++ {
			cmd := commands[i]
			switch strings.ToUpper(cmd) {
			case "SET":
				Set(i, commands)
			}
		}
	}
}
