package sshconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxIncludeDepth = 32

// ParseFileExpanded reads path and recursively resolves Include directives.
// Files are visited in source order and all returned records retain provenance.
func ParseFileExpanded(path string) (*Config, error) {
	config := &Config{Path: path}
	active := make(map[string]bool)
	if err := resolveFile(path, 0, active, config, "", 0); err != nil {
		return nil, err
	}
	return config, nil
}

func resolveFile(path string, depth int, active map[string]bool, result *Config, includePath string, includeLine int) error {
	canonical := canonicalPath(path)
	if active[canonical] {
		result.addDiagnosticAt(includeLine, fmt.Sprintf("include cycle detected for %s", path), includePath)
		return nil
	}
	if depth > maxIncludeDepth {
		result.addDiagnosticAt(includeLine, fmt.Sprintf("maximum Include depth %d exceeded at %s", maxIncludeDepth, path), includePath)
		return nil
	}

	fileConfig, entries, err := parseFileWithEntries(path)
	if err != nil {
		if depth == 0 {
			return err
		}
		result.addDiagnosticAt(includeLine, err.Error(), includePath)
		return nil
	}

	active[canonical] = true
	defer delete(active, canonical)
	result.Diagnostics = append(result.Diagnostics, fileConfig.Diagnostics...)

	for _, entry := range entries {
		switch {
		case entry.host != nil:
			result.Hosts = append(result.Hosts, *entry.host)
		case entry.include != nil:
			include := *entry.include
			result.Includes = append(result.Includes, include)
			if !entry.expandInclude {
				continue
			}
			matches, err := expandIncludePatterns(include)
			if err != nil {
				result.addDiagnosticAt(include.Line, err.Error(), include.Path)
				continue
			}
			if len(matches) == 0 {
				result.addDiagnosticAt(include.Line, fmt.Sprintf("Include pattern matched no files: %s", include.Pattern), include.Path)
				continue
			}
			for _, match := range matches {
				if err := resolveFile(match, depth+1, active, result, include.Path, include.Line); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func expandIncludePatterns(include Include) ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot expand Include %q: %w", include.Pattern, err)
	}

	var matches []string
	for _, rawPattern := range strings.Fields(include.Pattern) {
		pattern := strings.Trim(rawPattern, `"'`)
		if strings.HasPrefix(pattern, "~/") {
			pattern = filepath.Join(home, pattern[2:])
		} else if !filepath.IsAbs(pattern) {
			pattern = filepath.Join(filepath.Dir(include.Path), pattern)
		}

		found, err := filepath.Glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid Include pattern %q: %w", rawPattern, err)
		}
		matches = append(matches, found...)
	}
	return matches, nil
}

func canonicalPath(path string) string {
	absPath, err := filepath.Abs(path)
	if err == nil {
		path = absPath
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return filepath.Clean(path)
}

func (c *Config) addDiagnosticAt(line int, message, path string) {
	c.Diagnostics = append(c.Diagnostics, Diagnostic{File: path, Line: line, Message: message})
}
