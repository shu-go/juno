package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// errorLogFile is the path given via --log, -l. When empty, errors are
// written to stderr instead.
var errorLogFile string

// logError reports a runtime error (JSON parse failures, log/rule/notify
// errors, etc.) as a single JSONL entry: {"timestamp":"<RFC3339Nano>","message":"..."}.
// It's appended to errorLogFile when set, or written to stderr otherwise.
func logError(format string, args ...any) {
	entry := struct {
		Timestamp string `json:"timestamp"`
		Message   string `json:"message"`
	}{
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Message:   fmt.Sprintf(format, args...),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	data = append(data, '\n')

	if errorLogFile == "" {
		os.Stderr.Write(data)
		return
	}

	f, err := os.OpenFile(errorLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		os.Stderr.Write(data)
		fmt.Fprintf(os.Stderr, "juno: open log %s: %v\n", errorLogFile, err)
		return
	}
	defer f.Close()
	f.Write(data)
}
