package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

func Ping() string {
	return "+PONG\r\n"
}

func Echo(i int, commands []string) (string, int) {
	var response string
	if i < len(commands)-1 {
		response = fmt.Sprintf("$%d\r\n%s\r\n", len(commands[i+1]), commands[i+1])
		i++
	} else {
		response = "-ERR wrong number of arguments for 'echo' command\r\n"
	}
	return response, i
}

func Set(i int, commands []string) (string, int) {
	var response string
	if i < len(commands)-2 {
		key := commands[i+1]
		value := commands[i+2]
		KeyValuePairs[key] = value
		fmt.Printf("SET %s %s\n", key, value)

		if i+3 < len(commands) && strings.ToUpper(commands[i+3]) == "PX" {
			expiry, err := strconv.Atoi(commands[i+4])
			if err != nil {
				response = "-ERR invalid expiry\r\n"
				return response, i
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
	return response, i
}

func Get(i int, commands []string) (string, int) {
	var response string
	if i < len(commands)-1 {
		key := commands[i+1]
		fmt.Printf("GET %s\n", key)
		if value, ok := KeyValuePairs[key]; ok {
			if expiry, found := KeyExpiryTime[key]; found && expiry <= time.Now().UnixNano() {
				// Key has expired, delete it
				delete(KeyValuePairs, key)
				delete(KeyExpiryTime, key)
				response = EmptyResponse()
			} else {
				response = fmt.Sprintf("$%d\r\n%s\r\n", len(value), value)
			}
			i++
		} else {
			response = EmptyResponse()
			i++
		}
	} else {
		response = "-ERR wrong number of arguments for 'get' command\r\n"
	}
	return response, i
}

func Info(i int, commands []string) (string, int) {
	var response string
	if i < len(commands)-1 && strings.ToUpper(commands[i+1]) == "REPLICATION" {
		if isReplica {
			response = "$10\r\nrole:slave\r\n"
		} else {
			raw := fmt.Sprintf("role:master\nmaster_replid:%s\nmaster_repl_offset:%d", masterReplID, masterReplOffset)
			response = fmt.Sprintf("$%d\r\n%s\r\n", len(raw), raw)
		}
		i++
	} else {
		response = "$13\r\n# Replication\r\n"
	}
	return response, i
}

// HandleREPLCONF handles the REPLCONF command for replica configuration
func HandleREPLCONF(index int, commands []string) (string, int) {
	var response string
	if index < len(commands)-1 {
		subCommand := strings.ToUpper(commands[index+1])
		switch subCommand {
		case "LISTENING-PORT":
			if index < len(commands)-2 {
				rport, err := strconv.Atoi(commands[index+2])
				if err != nil {
					response = "-ERR invalid listening port\r\n"
				} else {
					replicaPort = rport
					response = "+OK\r\n"
					index += 2
				}
			} else {
				response = "-ERR wrong number of arguments for 'REPLCONF' command\r\n"
			}
		case "CAPA":
			if index < len(commands)-2 {
				capability := strings.ToUpper(commands[index+2])
				if capability == "PSYNC2" {
					response = "+OK\r\n"
				} else {
					response = "-ERR unsupported capability\r\n"
				}
				index += 2
			} else {
				response = "-ERR wrong number of arguments for 'REPLCONF' command\r\n"
			}
		default:
			response = "-ERR unknown subcommand for 'REPLCONF' command\r\n"
		}
	} else {
		response = "-ERR wrong number of arguments for 'REPLCONF' command\r\n"
	}
	return response, index
}

// Psync handles the PSYNC command.
func Psync(i int) (string, int) {
	return fmt.Sprintf("+FULLRESYNC %s %d\r\n", masterReplID, masterReplOffset), i + 2
}

// SendEmptyRDBFile sends an empty RDB file to the replica.
func SendEmptyRDBFile(conn net.Conn) []byte {
	var data []byte
	fmt.Println("Sending empty RDB file to replica")

	emptyRDBHex := "524544495330303131fa0972656469732d76657205372e322e30fa0a72656469732d62697473c040fa056374696d65c26d08bc65fa08757365642d6d656dc2b0c41000fa08616f662d62617365c000fff06e3bfec0ff5aa2"
	emptyRDBBytes, err := hex.DecodeString(emptyRDBHex)
	if err != nil {
		fmt.Println("Error decoding empty RDB hex:", err)
		return data
	}

	rdbLength := len(emptyRDBBytes)
	lengthPrefix := "$" + strconv.Itoa(rdbLength) + "\r\n"
	data = append([]byte(lengthPrefix), emptyRDBBytes...)
	return data
}

func PropagateToReplicas(replConnections []net.Conn, commands []string) {
	command := "*" + strconv.Itoa(len(commands)) + "\r\n"
	for _, c := range commands {
		command = command + "$" + strconv.Itoa(len(c)) + "\r\n" + c + "\r\n"
	}
	for _, r := range replicas {
		replWriter := bufio.NewWriter(r)
		err := WriteResponse(replWriter, command)
		if err != nil {
			fmt.Println("Error propagating to replica", err)
		}
		fmt.Println("Sent command", command, "to replica", r.RemoteAddr())
	}
}
