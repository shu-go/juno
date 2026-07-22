package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the content of juno.yaml.
type Config struct {
	Rules   string `yaml:"rules"`
	Command string `yaml:"command"`

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

	configDir := filepath.Dir(path)
	if filepath.IsAbs(cfg.Rules) {
		cfg.RulesDir = cfg.Rules
	} else {
		cfg.RulesDir = filepath.Join(configDir, cfg.Rules)
	}

	return &cfg, nil
}
