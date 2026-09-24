package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	gli "github.com/shu-go/gli/v2"
)

// verbose prints progress messages when --verbose, -v is given. It is a
// no-op otherwise.
var verbose = func(format string, args ...any) {}

type globalCmd struct {
	Config  string `cli:"config,c" help:"path to the config file (default: juno.yaml next to the executable)"`
	Rule    string `cli:"rule,r" help:"only process rule files whose path contains this substring"`
	Verbose bool   `cli:"verbose,v" help:"print the config file, rule files, and log files as they are processed"`
	DryRun  bool   `cli:"dry-run" help:"ignore Command and print notification content to stdout instead; don't update Latest"`
	Log     string `cli:"log,l" help:"append runtime errors (JSONL) to this file instead of stderr"`

	Generate generateCmd `cli:"generate" help:"generate a config or rule file template"`
}

func (g *globalCmd) Init() {
	if g.Config == "" {
		g.Config = "juno.yaml"
		if exe, err := os.Executable(); err == nil {
			g.Config = filepath.Join(filepath.Dir(exe), "juno.yaml")
		}
	}
}

func (g *globalCmd) Before() error {
	if g.Verbose {
		verbose = func(format string, args ...any) {
			fmt.Fprintf(os.Stdout, format, args...)
		}
	}
	errorLogFile = g.Log
	return nil
}

func (g *globalCmd) Run() error {
	verbose("config: %s\n", g.Config)

	cfg, err := LoadConfig(g.Config)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	rules, err := LoadRules(cfg.RulesDir)
	if err != nil {
		return fmt.Errorf("load rules: %w", err)
	}

	if g.Rule != "" {
		filtered := rules[:0]
		for _, rule := range rules {
			if strings.Contains(rule.FilePath, g.Rule) {
				filtered = append(filtered, rule)
			}
		}
		rules = filtered
	}

	for _, rule := range rules {
		verbose("rule: %s\n", rule.FilePath)
		if err := ProcessRule(rule, []string(cfg.Command), time.Duration(cfg.Interval), g.DryRun); err != nil {
			logError("rule %s: %v", rule.Name, err)
		}
	}

	return nil
}

func main() {
	g := &globalCmd{}
	app := gli.NewWith(g)
	app.Name = "juno"
	app.Version = "0.1.0"
	app.SuppressErrorOutput = true

	if err := app.Run(os.Args); err != nil {
		logError("%v", err)
		os.Exit(1)
	}
}
