package main

import (
	"fmt"
	"os"
	"path/filepath"
)

type generateCmd struct {
	Config string `cli:"config,c=FILE_PATH" help:"create a config file template at FILE_PATH"`
	Rule   string `cli:"rule,r=FILE_PATH" help:"create a rule file template at FILE_PATH"`
	Force  bool   `cli:"force,f" help:"overwrite FILE_PATH if it already exists"`
}

const configTemplate = `# juno config file
# https://github.com/shu-go/juno

# rules: folder containing rule files (searched recursively). Optional, default: ./rules/
rules: ./rules/

# command: command line(s) invoked for each notification; the formatted
# content is sent to each one's standard input. A single string or a list of
# strings; when a list, the same content is sent to every command. Paths may
# be written with "/"; they are converted to the OS path separator at
# runtime.
command: "your-notify-command"

# interval: wait time after each notify command execution. Optional, default: 1s.
#interval: 1s
`

const ruleTemplate = `# juno rule file

# name: optional. Defaults to <folder>/<filename without extension>, relative
# to the rules folder.
#name: my-rule

# logs: glob pattern(s) of target log files, resolved relative to this rule
# file's directory. A single string or a list of strings.
logs:
  - "../logs/*.jsonl"

# encoding: character encoding of the log files above. Optional, default: utf8.
# One of: utf8, utf8bom, sjis, utf16le, utf16be
#encoding: utf8

# filter: Expr expression evaluated per log entry. Log entry fields are
# available as log.KEY_NAME. Optional, default: "true" (every entry is notified).
#   nil or false -> the entry is skipped
#   any other value -> the entry is notified
filter: "log.Level == \"error\""

# key: optional Expr expression producing a grouping key (log entry fields
# available as log.KEY_NAME) for entries that pass filter. Entries are
# grouped by this value and only the last one per group is notified. Omit to
# notify every matching entry immediately.
#key: "log.Level"

# notify: optional Expr expression producing the notification content as a
# string (log entry fields available as log.KEY_NAME). Omit to use the
# default "[name] {json}" format.
#notify: "log.Message"

# command: optional override of the config file's command for this rule. A
# single string or a list of strings, like logs above.
#command: "your-notify-command"

# latest: managed by juno. Do not edit manually.
#latest: ""
`

func (c *generateCmd) Run() error {
	if c.Config == "" && c.Rule == "" {
		return fmt.Errorf("generate: specify --config FILE_PATH and/or --rule FILE_PATH")
	}

	if c.Config != "" {
		if err := writeTemplate(c.Config, configTemplate, c.Force); err != nil {
			return fmt.Errorf("generate: config: %w", err)
		}
		fmt.Fprintf(os.Stdout, "generated config template: %s\n", c.Config)
	}

	if c.Rule != "" {
		if err := writeTemplate(c.Rule, ruleTemplate, c.Force); err != nil {
			return fmt.Errorf("generate: rule: %w", err)
		}
		fmt.Fprintf(os.Stdout, "generated rule template: %s\n", c.Rule)
	}

	return nil
}

func writeTemplate(path, content string, force bool) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL
	if force {
		flags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	}

	f, err := os.OpenFile(path, flags, 0644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("%s already exists", path)
		}
		return err
	}
	defer f.Close()

	_, err = f.WriteString(content)
	return err
}
