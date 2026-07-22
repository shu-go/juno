package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Notify runs the configured notify command line, sending content to its
// standard input.
func Notify(commandLine, content string) error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", commandLine)
	} else {
		cmd = exec.Command("sh", "-c", commandLine)
	}
	cmd.Stdin = strings.NewReader(content)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("notify command failed: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
