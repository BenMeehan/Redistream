package main

import (
	"fmt"
	"math"
	"slices"
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

func (s *stream) splitID(id string) (millisecondsTime, sequenceNumber uint64, hasSequence bool, err error) {
	parts := strings.Split(id, "-")
	millisecondsTime, err = strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return
	}
	if len(parts) > 1 {
		sequenceNumber, err = strconv.ParseUint(parts[1], 10, 64)
		hasSequence = true
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

func searchStreamEntries(entries []*streamEntry, targetMs, targetSeq uint64, lo, hi int) int {
	for lo <= hi {
		mid := (lo + hi) / 2
		entry := entries[mid]
		if targetMs == entry.id[0] && targetSeq == entry.id[1] {
			lo = mid
			break
		} else if targetMs == entry.id[0] && entry.id[1] > targetSeq {
			hi = mid - 1
		} else if targetMs == entry.id[0] && entry.id[1] < targetSeq {
			lo = mid + 1
		} else if targetMs < entry.id[0] {
			hi = mid - 1
		} else {
			lo = mid + 1
		}
	}
	return lo
}

func (srv *serverState) handleStreamRange(streamKey, start, end string) (response string) {

	stream, exists := srv.streams[streamKey]
	if !exists || len(stream.entries) == 0 {
		response = "*0\r\n"
		return
	}

	var startIndex, endIndex int

	if start == "-" {
		startIndex = 0
	} else {
		startMs, startSeq, startHasSeq, _ := stream.splitID(start)
		if !startHasSeq {
			startSeq = 0
		}
		startIndex = searchStreamEntries(stream.entries, startMs, startSeq, 0, len(stream.entries)-1)
	}

	if end == "+" {
		endIndex = len(stream.entries) - 1
	} else {
		endMs, endSeq, endHasSeq, _ := stream.splitID(end)
		if !endHasSeq {
			endSeq = math.MaxUint64
		}
		endIndex = searchStreamEntries(stream.entries, endMs, endSeq, startIndex, len(stream.entries)-1)
		if endIndex >= len(stream.entries) {
			endIndex = len(stream.entries) - 1
		}
	}

	entriesCount := endIndex - startIndex + 1
	response = fmt.Sprintf("*%d\r\n", entriesCount)
	for index := startIndex; index <= endIndex; index++ {
		entry := stream.entries[index]
		id := fmt.Sprintf("%d-%d", entry.id[0], entry.id[1])
		response += fmt.Sprintf("*2\r\n$%d\r\n%s\r\n", len(id), id)
		response += fmt.Sprintf("*%d\r\n", len(entry.store))
		for _, kv := range entry.store {
			response += encodeBulkString(kv)
		}
	}

	return
}

func (srv *serverState) handleStreamRead(cmd []string) (response string) {
	readKeyIndex := 2

	isBlocking := false
	blockTimeout := 0
	if cmd[1] == "block" {
		isBlocking = true
		blockTimeout, _ = strconv.Atoi(cmd[2])
		readKeyIndex += 2
	}
	_ = blockTimeout

	readCount := (len(cmd) - readKeyIndex) / 2
	readStartIndex := readKeyIndex + readCount

	readParams := []struct{ key, start string }{}
	for i := 0; i < readCount; i++ {
		streamKey := cmd[i+readKeyIndex]
		start := cmd[i+readStartIndex]

		_, exists := srv.streams[streamKey]
		if exists {
			readParams = append(readParams, struct{ key, start string }{streamKey, start})
		}
	}

	response = fmt.Sprintf("*%d\r\n", len(readParams))

	for _, readParam := range readParams {

		streamKey, start := readParam.key, readParam.start

		// stream key + entry (below)
		response += fmt.Sprintf("*%d\r\n", 2)
		response += encodeBulkString(streamKey)

		stream := srv.streams[streamKey]

		var startMs, startSeq uint64
		var startHasSeq bool

		if start == "$" {
			startMs, startSeq = stream.last[0], stream.last[1]
		} else {
			startMs, startSeq, startHasSeq, _ = stream.splitID(start)
			if !startHasSeq {
				startSeq = 0
			}
		}

		var entry *streamEntry
		var startIndex int

		for entry == nil {
			startIndex = searchStreamEntries(stream.entries, startMs, startSeq, startIndex, len(stream.entries)-1)

			if startIndex < len(stream.entries) {
				entry = stream.entries[startIndex]
			}

			// if found exact match, need to get the next one (xread bound is exclusive)
			if entry != nil && entry.id[0] == startMs && entry.id[1] == startSeq {
				if startIndex+1 < len(stream.entries) {
					entry = stream.entries[startIndex+1]
				} else {
					entry = nil
				}
			}

			if entry == nil {
				if isBlocking {
					waitForAdd := make(chan bool)
					stream.blocked = append(stream.blocked, &waitForAdd)
					timedOut := false
					if blockTimeout > 0 {
						fmt.Printf("Waiting for a write on stream %s (timeout = %d ms)...\n", streamKey, blockTimeout)
						timer := time.After(time.Duration(blockTimeout) * time.Millisecond)
						select {
						case <-waitForAdd:
							timedOut = false
						case <-timer:
							timedOut = true
						}
					} else {
						fmt.Printf("Waiting for a write on stream %s (no timeout!)...\n", streamKey)
						<-waitForAdd
					}
					stream.blocked = slices.DeleteFunc(stream.blocked, func(ch *chan bool) bool { return ch == &waitForAdd })
					if timedOut {
						response = "$-1\r\n"
						return
					}
				} else {
					break
				}
			}
		}

		if entry == nil {
			response = "*0\r\n"
			return
		}

		// single entry
		response += "*1\r\n"
		id := fmt.Sprintf("%d-%d", entry.id[0], entry.id[1])
		response += fmt.Sprintf("*2\r\n$%d\r\n%s\r\n", len(id), id)
		response += fmt.Sprintf("*%d\r\n", len(entry.store))
		for _, kv := range entry.store {
			response += encodeBulkString(kv)
		}
	}
	return
}
