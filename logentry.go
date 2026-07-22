package main

import (
	"bufio"
	"encoding/json"
	"io"
)

// LogEntry is a single JSONL log entry.
type LogEntry map[string]any

func (e LogEntry) Timestamp() string {
	v, _ := e["timestamp"].(string)
	return v
}

// ReadLogEntries reads JSONL from r, calling fn for each successfully parsed entry
// in file order. Lines that fail to parse as a JSON object are skipped.
func ReadLogEntries(r io.Reader, fn func(LogEntry) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry LogEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		if err := fn(entry); err != nil {
			return err
		}
	}
	return scanner.Err()
}
