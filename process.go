package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// ProcessRule runs one rule to completion: scanning its log files, judging
// and notifying, then persisting the new Latest value. When dryRun is true,
// Command is ignored and notification content is printed to stdout (along
// with the rule and log file involved) instead, and Latest is not updated.
func ProcessRule(rule *Rule, configCommand string, dryRun bool) error {
	notifyCmd := rule.CommandLine(configCommand)

	logFiles, err := matchLogFiles(rule)
	if err != nil {
		return fmt.Errorf("resolve logs: %w", err)
	}

	pending := map[string]LogEntry{}      // Key result (as string) -> last matching entry
	pendingCount := map[string]int{}      // Key result (as string) -> number of matching entries
	pendingLogFile := map[string]string{} // Key result (as string) -> log file of last matching entry
	var pendingOrder []string             // insertion order of pending keys, for stable notify order

	for _, logFile := range logFiles {
		verbose("log: %s\n", logFile)

		f, err := os.Open(logFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "juno: rule %s: log %s: %v\n", rule.Name, logFile, err)
			continue
		}

		r, err := decodingReader(rule.Encoding, f)
		if err != nil {
			f.Close()
			fmt.Fprintf(os.Stderr, "juno: rule %s: log %s: %v\n", rule.Name, logFile, err)
			continue
		}

		err = ReadLogEntries(r, func(entry LogEntry) error {
			rule.NoteTimestamp(entry.Timestamp())

			if !rule.IsNew(entry.Timestamp()) {
				return nil
			}

			result, err := runProgram(rule.FilterProgram, entry)
			if err != nil {
				return fmt.Errorf("filter: %w", err)
			}

			if result == nil {
				return nil
			}
			if b, ok := result.(bool); ok && !b {
				return nil
			}

			if rule.KeyProgram == nil {
				content, err := formatNotification(rule, entry, 1)
				if err != nil {
					return fmt.Errorf("notify: %w", err)
				}
				deliver(rule, notifyCmd, logFile, content, dryRun)
				return nil
			}

			keyResult, err := runProgram(rule.KeyProgram, entry)
			if err != nil {
				return fmt.Errorf("key: %w", err)
			}

			key := fmt.Sprint(keyResult)
			if _, seen := pending[key]; !seen {
				pendingOrder = append(pendingOrder, key)
			}
			pending[key] = entry
			pendingCount[key]++
			pendingLogFile[key] = logFile
			return nil
		})
		f.Close()
		if err != nil {
			return fmt.Errorf("log %s: %w", logFile, err)
		}
	}

	for _, key := range pendingOrder {
		entry := pending[key]
		content, err := formatNotification(rule, entry, pendingCount[key])
		if err != nil {
			return fmt.Errorf("notify: %w", err)
		}
		deliver(rule, notifyCmd, pendingLogFile[key], content, dryRun)
	}

	if dryRun {
		return nil
	}

	if err := rule.SaveLatest(); err != nil {
		return fmt.Errorf("save latest: %w", err)
	}

	return nil
}

// deliver sends content to notifyCmd, or, when dryRun is true, prints it to
// stdout along with the rule and log file it came from instead of running
// notifyCmd.
func deliver(rule *Rule, notifyCmd, logFile, content string, dryRun bool) {
	if dryRun {
		fmt.Printf("[dry-run] rule=%s log=%s\n%s\n\n", rule.FilePath, logFile, content)
		return
	}
	if err := Notify(notifyCmd, content); err != nil {
		fmt.Fprintf(os.Stderr, "juno: rule %s: %v\n", rule.Name, err)
	}
}

// matchLogFiles resolves the rule's Logs glob patterns (relative to the
// rule file's directory) to a sorted, de-duplicated list of file paths.
func matchLogFiles(rule *Rule) ([]string, error) {
	seen := map[string]bool{}
	var files []string

	for _, pattern := range rule.Logs {
		joined := filepath.Join(rule.BaseDir, pattern)
		matches, err := doublestar.FilepathGlob(filepath.ToSlash(joined))
		if err != nil {
			return nil, fmt.Errorf("pattern %q: %w", pattern, err)
		}
		for _, m := range matches {
			if !seen[m] {
				seen[m] = true
				files = append(files, m)
			}
		}
	}

	sort.Strings(files)
	return files, nil
}

func runProgram(program *vm.Program, entry LogEntry) (any, error) {
	env := map[string]any{"log": map[string]any(entry)}
	return expr.Run(program, env)
}

func runNotifyProgram(program *vm.Program, entry LogEntry, count int, ruleName string) (any, error) {
	env := map[string]any{"log": map[string]any(entry), "Count": count, "Rule": ruleName}
	return expr.Run(program, env)
}

func formatNotification(rule *Rule, entry LogEntry, count int) (string, error) {
	if rule.NotifyProgram != nil {
		result, err := runNotifyProgram(rule.NotifyProgram, entry, count, rule.Name)
		if err != nil {
			return "", err
		}
		if s, ok := result.(string); ok {
			return s, nil
		}
		return fmt.Sprint(result), nil
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("[%s] %s", rule.Name, string(data)), nil
}
