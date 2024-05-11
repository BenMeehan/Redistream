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
	conn      net.Conn
	offset    int
	ackOffset int // TODO: keep track of this also
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
	masterConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", srv.config.replicaofHost, srv.config.replicaofPort))
	if err != nil {
		fmt.Printf("Failed to connect to master %v\n", err)
		os.Exit(1)
	}

	// TODO: check responses
	reader := bufio.NewReader(masterConn)
	masterConn.Write([]byte(encodeStringArray([]string{"PING"})))
	reader.ReadString('\n')
	masterConn.Write([]byte(encodeStringArray([]string{"REPLCONF", "listening-port", strconv.Itoa(srv.config.port)})))
	reader.ReadString('\n')
	masterConn.Write([]byte(encodeStringArray([]string{"REPLCONF", "capa", "psync2"})))
	reader.ReadString('\n')
	masterConn.Write([]byte(encodeStringArray([]string{"PSYNC", "?", "-1"})))
	reader.ReadString('\n')

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

	go srv.handlePropagation(reader, masterConn)
}

var emptyRDB = []byte("524544495330303131fa0972656469732d76657205372e322e30fa0a72656469732d62697473c040fa056374696d65c26d08bc65fa08757365642d6d656dc2b0c41000fa08616f662d62617365c000fff06e3bfec0ff5aa2")

func sendFullResynch(conn net.Conn) int {
	buffer := make([]byte, hex.DecodedLen(len(emptyRDB)))
	// TODO: check for errors
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
				// TODO: mutex?
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
		response, _ := srv.handleCommand(cmd)

		// REPLCONF ACK is the only response that a replica send back to master
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

func (srv *serverState) handleWait(count, timeout int) string {
	getAckCmd := []byte(encodeStringArray([]string{"REPLCONF", "GETACK", "*"}))

	acks := 0

	for i := 0; i < len(srv.replicas); i++ {
		if srv.replicas[i].offset > 0 {
			bytesWritten, _ := srv.replicas[i].conn.Write(getAckCmd)
			srv.replicas[i].offset += bytesWritten
			go func(conn net.Conn) {
				fmt.Println("waiting response from replica", conn.RemoteAddr().String())
				buffer := make([]byte, 1024)
				// TODO: Ignoring result, just "flushing" the response
				_, err := conn.Read(buffer)
				if err == nil {
					fmt.Println("got response from replica", conn.RemoteAddr().String())
				} else {
					fmt.Println("error from replica", conn.RemoteAddr().String(), " => ", err.Error())
				}
				srv.ackReceived <- true
			}(srv.replicas[i].conn)
		} else {
			acks++
		}
	}

	timer := time.After(time.Duration(timeout) * time.Millisecond)

outer:
	for acks < count {
		select {
		case <-srv.ackReceived:
			acks++
			fmt.Println("acks =", acks)
		case <-timer:
			fmt.Println("timeout! acks =", acks)
			break outer
		}
	}

	return encodeInt(acks)
}
