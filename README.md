# juno

A command-line tool that scans JSONL logs and sends notifications according to configurable rules.

## Features

- Reads JSONL log files and notifies on entries that match a rule
- Notification judgment and formatting are written as [Expr](https://github.com/expr-lang/expr) expressions
- Rule files are YAML; multiple rule files and subfolders are supported
- Duplicate notification suppression (entries are grouped and only the last one per group is notified)
- Per-rule log file character encoding (utf8, utf8bom, sjis, utf16le, utf16be)
- `--dry-run` prints notification content to stdout instead of running the notify command

## Install

```sh
go install github.com/shu-go/juno@latest
```

## Getting Started

juno only reads JSONL log files and runs a notify command on matching entries — it doesn't collect logs or send notifications by itself. Before running it, prepare the following:

1. **JSONL log files.** Either have your system emit them directly, or convert existing logs (plain text, CSV, etc.) into JSONL beforehand. Each line must be a JSON object with at least a `timestamp` field, which juno uses to avoid reprocessing the same entries on the next run.
2. **A rule file** describing which JSONL files to scan and how to judge/format notifications for them (see [Rule file](#rule-file-rulesyaml) below).
3. **A notify command** that reads the notification content from its standard input and delivers it wherever you want (Slack, email, a custom script, etc.).

With those in place:

```sh
# create starter templates
juno generate --config juno.yaml --rule rules/my-rule.yaml

# edit juno.yaml (set command) and rules/my-rule.yaml (set logs, filter, ...)

# run
juno --config juno.yaml
```

## Usage

### Generate config/rule file templates

```sh
juno generate --config juno.yaml --rule rules/my-rule.yaml
```

Fails if the target file already exists; pass `--force, -f` to overwrite it.

### Run

```sh
juno [--verbose, -v] [--dry-run] [--config, -c FILE_PATH] [--rule, -r FILE_PATH]
```

| Option | Description |
| --- | --- |
| `--config, -c` | Path to the config file (default: `juno.yaml` next to the executable) |
| `--rule, -r` | Only process rule files whose path contains this substring |
| `--verbose, -v` | Print the config file, rule files, and log files as they are processed |
| `--dry-run` | Don't run the notify command; print notification content to stdout instead (and don't update `latest`) |

## Config file (juno.yaml)

```yaml
# rules: folder containing rule files (default: ./rules/)
rules: ./rules/

# command: command line invoked for each notification (content is sent via stdin)
command: "your-notify-command"
```

## Rule file (rules/*.yaml)

```yaml
# logs: glob pattern(s) of target log files (a string or a list of strings)
logs:
  - "../logs/*.jsonl"

# encoding: character encoding of the log files (default: utf8)
#encoding: utf8

# filter: Expr expression deciding whether to notify (nil/false = skip, anything else = notify)
# Optional, default: "true" (every entry is notified)
filter: "log.Level == \"error\""

# key: optional Expr expression producing a grouping key for entries that pass
# filter; only the last entry per key is notified. Omit to notify every match immediately.
#key: "log.Level"

# notify: Expr expression formatting the notification content (default format if omitted)
#notify: "log.Message"

# command: overrides the config file's command for this rule
#command: "your-notify-command"
```
