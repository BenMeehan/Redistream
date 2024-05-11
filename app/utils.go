package main

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
)

func EmptyResponse() string {
	return fmt.Sprintf("$%d\r\n", -1)
}

// readCommand reads and parses a Redis command from the client connection.
func ReadCommand(reader *bufio.Reader) ([]string, string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, "", err
	}
	length, err := strconv.Atoi(strings.TrimSpace(line[1:]))
	if err != nil {
		return nil, "", err
	}

	commands := make([]string, 0)

	for i := 0; i < length; i++ {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, "", err
		}
		elementLength, err := strconv.Atoi(strings.TrimSpace(line[1:]))
		if err != nil {
			return nil, "", err
		}

		element := make([]byte, elementLength)
		_, err = reader.Read(element)
		if err != nil {
			return nil, "", err
		}
		commands = append(commands, string(element))

		_, err = reader.ReadString('\n')
		if err != nil {
			return nil, "", err
		}
	}

	return commands, line, nil
}

// writeResponse writes a response to the client connection.
func WriteResponse(writer *bufio.Writer, response string) error {
	_, err := writer.WriteString(response)
	if err != nil {
		return err
	}
	return writer.Flush()
}
