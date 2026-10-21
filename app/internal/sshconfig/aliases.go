package sshconfig

import (
	"path/filepath"
	"sort"
	"strings"
)

func inspectSSHAliases(content string) []Alias {
	global := make(map[string][]string)
	var blocks []sshHostBlock
	var current *sshHostBlock
	mode := "global"
	seen := make(map[string]struct{})
	var names []string
	for _, line := range strings.Split(content, "\n") {
		tokens := sshConfigTokens(line)
		if len(tokens) == 0 {
			continue
		}
		directive, arguments := configDirectiveArgs(tokens)
		if directive == "host" {
			mode = "host"
			current = &sshHostBlock{patterns: make([]string, 0, len(arguments)), options: make(map[string][]string)}
			for _, token := range arguments {
				pattern := token.value
				current.patterns = append(current.patterns, pattern)
				if strings.ContainsAny(pattern, "*!?") || strings.HasPrefix(pattern, "!") {
					continue
				}
				if _, found := seen[pattern]; !found {
					seen[pattern] = struct{}{}
					names = append(names, pattern)
				}
			}
			blocks = append(blocks, *current)
			continue
		}
		if directive == "match" {
			mode = "match"
			current = nil
			continue
		}
		if len(arguments) == 0 || directive == "include" {
			continue
		}
		var target map[string][]string
		switch mode {
		case "global":
			target = global
		case "host":
			if current != nil {
				current.options[directive] = append(current.options[directive], arguments[0].value)
				blocks[len(blocks)-1] = *current
			}
		}
		if target != nil {
			target[directive] = append(target[directive], arguments[0].value)
		}
	}

	aliases := make([]Alias, 0, len(names))
	for _, name := range names {
		options := make(map[string][]string)
		for key, values := range global {
			options[key] = append(options[key], values...)
		}
		for _, block := range blocks {
			if hostPatternsMatch(block.patterns, name) {
				for key, values := range block.options {
					options[key] = append(options[key], values...)
				}
			}
		}
		alias := Alias{Name: name}
		alias.HostName = firstOption(options, "hostname")
		alias.User = firstOption(options, "user")
		alias.Port = firstOption(options, "port")
		alias.Identity = firstOption(options, "identityfile")
		alias.IdentityCount = len(options["identityfile"])
		aliases = append(aliases, alias)
	}
	sort.SliceStable(aliases, func(i, j int) bool { return strings.ToLower(aliases[i].Name) < strings.ToLower(aliases[j].Name) })
	return aliases
}

type sshHostBlock struct {
	patterns []string
	options  map[string][]string
}

func hostPatternsMatch(patterns []string, alias string) bool {
	matched := false
	for _, pattern := range patterns {
		negative := strings.HasPrefix(pattern, "!")
		if negative {
			pattern = strings.TrimPrefix(pattern, "!")
		}
		ok, err := filepath.Match(strings.ToLower(pattern), strings.ToLower(alias))
		if err == nil && ok {
			if negative {
				return false
			}
			matched = true
		}
	}
	return matched
}

func firstOption(options map[string][]string, key string) string {
	if values := options[key]; len(values) > 0 {
		return values[0]
	}
	return ""
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
