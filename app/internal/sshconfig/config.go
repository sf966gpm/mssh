package sshconfig

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Field struct {
	Name  string
	Value string
	Line  int
}

func (f Field) DisplayValue() string {
	if value, err := strconv.Unquote(f.Value); err == nil {
		return value
	}
	return f.Value
}

type HostBlock struct {
	Patterns []string
	Line     int
	Fields   []Field
}

type Include struct {
	Pattern string
	Line    int
}

type Diagnostic struct {
	File    string
	Line    int
	Message string
}

type Config struct {
	Path        string
	Hosts       []HostBlock
	Includes    []Include
	Diagnostics []Diagnostic
}

var supportedFields = map[string]bool{
	"hostname":     true,
	"user":         true,
	"port":         true,
	"identityfile": true,
}

func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine current user's home directory: %w", err)
	}
	return filepath.Join(home, ".ssh", "config"), nil
}

func ParseFile(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("SSH config does not exist: %s", path)
		}
		return nil, fmt.Errorf("cannot read SSH config %s: %w", path, err)
	}
	defer file.Close()

	config := &Config{Path: path}
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	var current *HostBlock
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(stripComment(scanner.Text()))
		if line == "" {
			continue
		}

		name, value, ok := splitDirective(line)
		if !ok {
			config.addDiagnostic(lineNumber, "malformed directive")
			continue
		}
		lowerName := strings.ToLower(name)

		switch lowerName {
		case "host":
			patterns := strings.Fields(value)
			if len(patterns) == 0 {
				config.addDiagnostic(lineNumber, "Host requires at least one pattern")
				current = nil
				continue
			}
			config.Hosts = append(config.Hosts, HostBlock{Patterns: patterns, Line: lineNumber})
			current = &config.Hosts[len(config.Hosts)-1]
		case "include":
			if strings.TrimSpace(value) == "" {
				config.addDiagnostic(lineNumber, "Include requires a path or pattern")
				continue
			}
			config.Includes = append(config.Includes, Include{Pattern: strings.TrimSpace(value), Line: lineNumber})
		case "match":
			config.addDiagnostic(lineNumber, "unsupported directive Match; following fields are not attributed to a Host block")
			current = nil
		default:
			if !supportedFields[lowerName] {
				config.addDiagnostic(lineNumber, fmt.Sprintf("unsupported directive %q", name))
				continue
			}
			if strings.TrimSpace(value) == "" {
				config.addDiagnostic(lineNumber, fmt.Sprintf("%s requires a value", name))
				continue
			}
			if current == nil {
				config.addDiagnostic(lineNumber, fmt.Sprintf("directive %s is outside a supported Host block", name))
				continue
			}
			current.Fields = append(current.Fields, Field{Name: name, Value: value, Line: lineNumber})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("cannot read SSH config %s: %w", path, err)
	}
	return config, nil
}

func (c *Config) addDiagnostic(line int, message string) {
	c.Diagnostics = append(c.Diagnostics, Diagnostic{File: c.Path, Line: line, Message: message})
}

func splitDirective(line string) (string, string, bool) {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return "", "", false
	}
	name := parts[0]
	return name, strings.TrimSpace(line[len(name):]), true
}

func stripComment(line string) string {
	quoted := false
	escaped := false
	for i, char := range line {
		switch {
		case escaped:
			escaped = false
		case char == '\\':
			escaped = true
		case char == '"':
			quoted = !quoted
		case char == '#' && !quoted:
			return line[:i]
		}
	}
	return line
}
