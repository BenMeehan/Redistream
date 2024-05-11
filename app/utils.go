package main

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

func encodeBulkString(s string) string {
	if len(s) == 0 {
		return "$-1\r\n"
	}
	return fmt.Sprintf("$%d\r\n%s\r\n", len(s), s)
}

func encodeSimpleString(s string) string {
	return fmt.Sprintf("+%s\r\n", s)
}

func encodeStringArray(arr []string) string {
	result := fmt.Sprintf("*%d\r\n", len(arr))
	for _, s := range arr {
		result += encodeBulkString(s)
	}
	return result
}

func encodeInt(n int) string {
	return fmt.Sprintf(":%d\r\n", n)
}

func encodeError(e error) string {
	return fmt.Sprintf("-ERR %s\r\n", e.Error())
}

func decodeStringArray(reader *bufio.Reader) (arr []string, bytesRead int, err error) {
	var arrSize, strSize int
	for {
		var token string
		token, err = reader.ReadString('\n')
		if err != nil {
			return
		}
		// HACK: should count bytes properly?
		bytesRead += len(token)
		token = strings.TrimRight(token, "\r\n")
		// TODO: do proper RESP parsing!!!
		switch {
		case arrSize == 0 && token[0] == '*':
			arrSize, err = strconv.Atoi(token[1:])
			if err != nil {
				return
			}
		case strSize == 0 && token[0] == '$':
			strSize, err = strconv.Atoi(token[1:])
			if err != nil {
				return
			}
		default:
			if len(token) != strSize {
				fmt.Printf("[from master] Wrong string size - got: %d, want: %d\n", len(token), strSize)
				break
			}
			arrSize--
			strSize = 0
			arr = append(arr, token)
		}
		if arrSize == 0 {
			break
		}
	}
	return
}
