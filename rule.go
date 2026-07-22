package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"gopkg.in/yaml.v3"
)

// filterExprEnv is the shape exposed to Filter and Key expressions:
// log.KEY_NAME.
var filterExprEnv = map[string]any{"log": map[string]any{}}

// notifyExprEnv is the shape exposed to Notify expressions: log.KEY_NAME,
// Count (the number of log entries deduplicated into this notification), and
// Rule (the rule's Name).
var notifyExprEnv = map[string]any{"log": map[string]any{}, "Count": 0, "Rule": ""}

// StringList unmarshals a YAML scalar or sequence of strings into a []string.
type StringList []string

func (s *StringList) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		var str string
		if err := node.Decode(&str); err != nil {
			return err
		}
		*s = []string{str}
	case yaml.SequenceNode:
		var list []string
		if err := node.Decode(&list); err != nil {
			return err
		}
		*s = list
	default:
		return fmt.Errorf("logs: expected a string or a list of strings")
	}
	return nil
}

// RuleFile is the raw content of a rule YAML file.
type RuleFile struct {
	Name     string     `yaml:"name"`
	Logs     StringList `yaml:"logs"`
	Encoding string     `yaml:"encoding"`
	Latest   string     `yaml:"latest"`
	Filter   string     `yaml:"filter"`
	Key      string     `yaml:"key"`
	Notify   string     `yaml:"notify"`
	Command  string     `yaml:"command"`
}

// Rule is a loaded, validated rule ready for processing.
type Rule struct {
	RuleFile

	FilePath string // path to the rule file
	BaseDir  string // directory of the rule file; Logs patterns resolve against this

	FilterProgram *vm.Program
	KeyProgram    *vm.Program // nil if Key is empty; grouping key for deduplication
	NotifyProgram *vm.Program // nil if Notify is empty

	// Latest tracked across processing; written back to FilePath when non-empty
	// and changed.
	newLatest string
}

// LoadRules recursively finds rule files under rulesDir and loads them.
// Rule files with defects are skipped after printing a message to stderr.
func LoadRules(rulesDir string) ([]*Rule, error) {
	var files []string
	err := filepath.WalkDir(rulesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".yaml" || ext == ".yml" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan rules dir: %w", err)
	}
	sort.Strings(files)

	rules := make([]*Rule, 0, len(files))
	for _, f := range files {
		rule, err := loadRuleFile(rulesDir, f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "juno: rule %s: %v\n", f, err)
			continue
		}
		rules = append(rules, rule)
	}
	return rules, nil
}

func loadRuleFile(rulesDir, path string) (*Rule, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	var rf RuleFile
	if err := yaml.Unmarshal(data, &rf); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}

	if len(rf.Logs) == 0 {
		return nil, fmt.Errorf("logs is required")
	}
	if rf.Encoding != "" && !validEncodings[strings.ToLower(rf.Encoding)] {
		return nil, fmt.Errorf("encoding: invalid value %q", rf.Encoding)
	}
	if rf.Filter == "" {
		rf.Filter = "true"
	}

	rule := &Rule{
		RuleFile: rf,
		FilePath: path,
		BaseDir:  filepath.Dir(path),
	}

	if rule.Name == "" {
		rel, err := filepath.Rel(rulesDir, path)
		if err != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		rule.Name = strings.TrimSuffix(rel, filepath.Ext(rel))
	}

	filterProgram, err := expr.Compile(rf.Filter, expr.Env(filterExprEnv))
	if err != nil {
		return nil, fmt.Errorf("filter: %w", err)
	}
	rule.FilterProgram = filterProgram

	if rf.Key != "" {
		keyProgram, err := expr.Compile(rf.Key, expr.Env(filterExprEnv))
		if err != nil {
			return nil, fmt.Errorf("key: %w", err)
		}
		rule.KeyProgram = keyProgram
	}

	if rf.Notify != "" {
		notifyProgram, err := expr.Compile(rf.Notify, expr.Env(notifyExprEnv))
		if err != nil {
			return nil, fmt.Errorf("notify: %w", err)
		}
		rule.NotifyProgram = notifyProgram
	}

	if rule.Command != "" {
		rule.Command = filepath.FromSlash(rule.Command)
	}

	return rule, nil
}

// CommandLine returns the rule's Command override if set, otherwise
// configCommand.
func (r *Rule) CommandLine(configCommand string) string {
	if r.Command != "" {
		return r.Command
	}
	return configCommand
}

// IsNew reports whether ts is strictly newer than the rule's recorded Latest,
// by lexicographic string comparison.
func (r *Rule) IsNew(ts string) bool {
	if r.Latest == "" {
		return true
	}
	return ts > r.Latest
}

// NoteTimestamp records ts as a candidate for the new Latest value, keeping
// the lexicographically greatest one seen.
func (r *Rule) NoteTimestamp(ts string) {
	if ts == "" {
		return
	}
	if ts > r.newLatest {
		r.newLatest = ts
	}
}

// SaveLatest writes the tracked newLatest value back into the rule file, if
// it advanced past the value the rule was loaded with.
func (r *Rule) SaveLatest() error {
	if r.newLatest == "" || r.newLatest <= r.Latest {
		return nil
	}

	data, err := os.ReadFile(r.FilePath)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	if len(doc.Content) == 0 {
		return fmt.Errorf("empty rule file")
	}
	mapping := doc.Content[0]
	if mapping.Kind != yaml.MappingNode {
		return fmt.Errorf("rule file is not a YAML mapping")
	}

	setMappingString(mapping, "latest", r.newLatest)

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if err := os.WriteFile(r.FilePath, out, 0644); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	r.Latest = r.newLatest
	return nil
}

// setMappingString sets key to value within a YAML mapping node, adding the
// key if it is not already present.
func setMappingString(mapping *yaml.Node, key, value string) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content[i+1].SetString(value)
			return
		}
	}
	mapping.Content = append(mapping.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
	)
}
