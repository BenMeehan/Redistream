package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

type stream struct {
	first   [2]uint64
	last    [2]uint64
	entries []*streamEntry
	blocked []*chan bool
}

type streamEntry struct {
	id    [2]uint64
	store []string
}

func newStream() *stream {
	return &stream{
		first:   [2]uint64{0, 0},
		last:    [2]uint64{0, 0},
		entries: make([]*streamEntry, 0),
		blocked: make([]*chan bool, 0),
	}
}

func (s *stream) addStreamEntry(id string) (*streamEntry, error) {
	millisecondsTime, sequenceNumber, err := s.getNextID(id)
	if err != nil {
		return nil, err
	}

	if s.first[0] == 0 && s.first[1] == 0 {
		s.first[0], s.first[1] = millisecondsTime, sequenceNumber
	}
	s.last[0], s.last[1] = millisecondsTime, sequenceNumber

	entry := new(streamEntry)
	entry.id[0] = millisecondsTime
	entry.id[1] = sequenceNumber
	entry.store = make([]string, 0)
	s.entries = append(s.entries, entry)
	return entry, nil
}

func (s *stream) getNextID(id string) (millisecondsTime, sequenceNumber uint64, err error) {
	parts := strings.Split(id, "-")

	if len(parts) == 1 && parts[0] == "*" {
		millisecondsTime = uint64(time.Now().UnixMilli())
		if millisecondsTime == s.last[0] {
			sequenceNumber = s.last[1] + 1
		}
	} else if len(parts) == 2 && parts[1] == "*" {
		millisecondsTime, _ = strconv.ParseUint(parts[0], 10, 64)
		if millisecondsTime == s.last[0] {
			sequenceNumber = s.last[1] + 1
		} else if millisecondsTime > s.last[0] {
			sequenceNumber = 0
		} else {
			return 0, 0, fmt.Errorf("the ID specified in XADD is equal or smaller than the target stream top item")
		}
	} else {
		millisecondsTime, _ = strconv.ParseUint(parts[0], 10, 64)
		sequenceNumber, _ = strconv.ParseUint(parts[1], 10, 64)
	}

	if millisecondsTime == 0 && sequenceNumber == 0 {
		return 0, 0, fmt.Errorf("The ID specified in XADD must be greater than 0-0")
	}

	if millisecondsTime < s.last[0] || millisecondsTime == s.last[0] && sequenceNumber <= s.last[1] {
		return 0, 0, fmt.Errorf("The ID specified in XADD is equal or smaller than the target stream top item")
	}

	return
}

func (srv *serverState) handleStreamAdd(streamKey, id string, kvpairs []string) (response string) {
	stream, exists := srv.streams[streamKey]
	if !exists {
		stream = newStream()
		srv.streams[streamKey] = stream
	}

	entry, err := stream.addStreamEntry(id)
	if err != nil {
		response = encodeError(err)
	} else {
		for i := 0; i < len(kvpairs); i += 2 {
			key, value := kvpairs[i], kvpairs[i+1]
			entry.store = append(entry.store, key, value)
		}
		response = encodeBulkString(fmt.Sprintf("%d-%d", entry.id[0], entry.id[1]))
	}

	for _, ch := range stream.blocked {
		*ch <- true
	}

	return
}
