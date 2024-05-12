package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type replica struct {
	conn   net.Conn
	offset int
}

func randReplid() string {
	chars := []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	result := make([]byte, 40)
	for i := range result {
		c := rand.Intn(len(chars))
		result[i] = chars[c]
	}
	return string(result)
}

func (srv *serverState) replicaHandshake() {
	masterConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", srv.config.masterHost, srv.config.masterPort))
	if err != nil {
		fmt.Printf("Failed to connect to master %v\n", err)
		os.Exit(1)
	}

	reader := bufio.NewReader(masterConn)
	masterConn.Write([]byte(encodeStringArray([]string{"PING"})))
	reader.ReadString('\n')
	masterConn.Write([]byte(encodeStringArray([]string{"REPLCONF", "listening-port", strconv.Itoa(srv.config.port)})))
	reader.ReadString('\n')
	masterConn.Write([]byte(encodeStringArray([]string{"REPLCONF", "capa", "psync2"})))
	reader.ReadString('\n')
	masterConn.Write([]byte(encodeStringArray([]string{"PSYNC", "?", "-1"})))
	reader.ReadString('\n')

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

	go srv.handlePropagation(reader, masterConn)

	srv.requestAcknowledgement()
}

var emptyRDB = []byte("524544495330303131fa0972656469732d76657205372e322e30fa0a72656469732d62697473c040fa056374696d65c26d08bc65fa08757365642d6d656dc2b0c41000fa08616f662d62617365c000fff06e3bfec0ff5aa2")

func sendFullResynch(conn net.Conn) int {
	buffer := make([]byte, hex.DecodedLen(len(emptyRDB)))
	hex.Decode(buffer, emptyRDB)
	conn.Write([]byte(fmt.Sprintf("$%d\r\n", len(buffer))))
	conn.Write(buffer)
	return len(buffer)
}

func (srv *serverState) propagateToReplicas(cmd []string) {
	if len(srv.replicas) == 0 {
		return
	}
	fmt.Printf("Propagating = %q\n", cmd)
	for i := 0; i < len(srv.replicas); i++ {
		fmt.Printf("Replicating to: %s\n", srv.replicas[i].conn.RemoteAddr().String())
		bytesWritten, err := srv.replicas[i].conn.Write([]byte(encodeStringArray(cmd)))
		// remove stale replicas
		if err != nil {
			fmt.Printf("Disconnected: %s\n", srv.replicas[i].conn.RemoteAddr().String())
			if len(srv.replicas) > 0 {
				last := len(srv.replicas) - 1
				srv.replicas[i] = srv.replicas[last]
				srv.replicas = srv.replicas[:last]
				i--
			}
		}
		srv.replicas[i].offset += bytesWritten
	}
}

func (srv *serverState) handlePropagation(reader *bufio.Reader, masterConn net.Conn) {
	defer masterConn.Close()

	for {
		cmd, cmdSize, err := decodeStringArray(reader)
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("Error decoding command from master: %v\n", err.Error())
		}

		if len(cmd) == 0 {
			break
		}

		fmt.Printf("[from master] Command = %q\n", cmd)
		response, _ := srv.handleCommand(cmd, cmdSize)

		if strings.ToUpper(cmd[0]) == "REPLCONF" {
			_, err := masterConn.Write([]byte(response))
			if err != nil {
				fmt.Printf("Error responding to master: %v\n", err.Error())
				break
			}
		}
		srv.replicaOffset += cmdSize
	}
}

func (srv *serverState) requestAcknowledgement() {
	cmd := encodeStringArray([]string{"REPLCONF", "GETACK", "*"})
	for _, r := range srv.replicas {
		reader := bufio.NewReader(r.conn)
		r.conn.Write([]byte(cmd))
		resp, _, _ := decodeStringArray(reader)
		offset, _ := strconv.Atoi(resp[2])
		r.offset += offset
	}
}

func (srv *serverState) waitForWriteAck(minReplicas int, t int) string {
	timer := time.After(time.Duration(t) * time.Second)
	cmd := encodeStringArray([]string{"REPLCONF", "GETACK", "*"})
	noOfAcks := 0

	for _, r := range srv.replicas {
		if r.offset > 0 {
			bytesWritten, err := r.conn.Write([]byte(cmd))
			if err != nil {
				fmt.Println("error from replica write", r.conn.RemoteAddr().String(), " => ", err.Error())
			}
			r.offset += bytesWritten
			go func(conn net.Conn) {
				fmt.Println("waiting response from replica", conn.RemoteAddr().String())
				buffer := make([]byte, 1024)
				_, err := conn.Read(buffer)
				if err == nil {
					fmt.Println("got response from replica", conn.RemoteAddr().String())
				} else {
					fmt.Println("error from replica", conn.RemoteAddr().String(), " => ", err.Error())
				}
				srv.ackReceived <- true
			}(r.conn)
		} else {
			noOfAcks++
		}
	}

outer:
	for noOfAcks < minReplicas {
		select {
		case <-srv.ackReceived:
			noOfAcks++
		case <-timer:
			fmt.Println("timed out")
			break outer
		}
	}

	return encodeInteger(noOfAcks)
}
