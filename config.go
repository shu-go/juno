package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// defaultInterval is the wait time after running the notify command when
// Interval is not set in the config file.
const defaultInterval = 1 * time.Second

// Duration unmarshals a YAML duration string (e.g. "1s", "500ms") into a
// time.Duration.
type Duration time.Duration

func (d *Duration) UnmarshalYAML(node *yaml.Node) error {
	var s string
	if err := node.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("interval: %w", err)
	}
	*d = Duration(parsed)
	return nil
}

// Config is the content of juno.yaml.
type Config struct {
	Rules    string   `yaml:"rules"`
	Command  string   `yaml:"command"`
	Interval Duration `yaml:"interval"`

	// RulesDir is Rules resolved to an absolute path (relative to the config
	// file's directory when Rules is a relative path).
	RulesDir string `yaml:"-"`
}

// LoadConfig reads and parses the config file at path.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Command == "" {
		return nil, fmt.Errorf("config: command is required")
	}
	cfg.Command = filepath.FromSlash(cfg.Command)

	if cfg.Rules == "" {
		cfg.Rules = "./rules/"
	}

	if cfg.Interval == 0 {
		cfg.Interval = Duration(defaultInterval)
	}

	configDir := filepath.Dir(path)
	if filepath.IsAbs(cfg.Rules) {
		cfg.RulesDir = cfg.Rules
	} else {
		cfg.RulesDir = filepath.Join(configDir, cfg.Rules)
	}

	return &cfg, nil
}
