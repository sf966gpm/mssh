package sshconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func sshConfigIncludes(content, sshDir string) ([]string, error) {
	home, localUser, localHost, err := sshIncludeExpansionContext()
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, line := range strings.Split(content, "\n") {
		tokens := sshConfigTokens(line)
		if len(tokens) == 0 {
			continue
		}
		directive, args := configDirectiveArgs(tokens)
		if directive != "include" {
			continue
		}
		for _, arg := range args {
			path, err := expandSSHIncludePattern(arg.value, home, localUser, localHost)
			if err != nil {
				return nil, fmt.Errorf("Include %q: %w", arg.value, err)
			}
			if !filepath.IsAbs(path) {
				path = filepath.Join(sshDir, path)
			}
			matches, err := filepath.Glob(path)
			if err != nil {
				return nil, fmt.Errorf("неверный шаблон Include %q: %w", arg.value, err)
			}
			for _, match := range matches {
				paths = appendUnique(paths, filepath.Clean(match))
			}
		}
	}
	sort.Slice(paths, func(i, j int) bool { return paths[i] < paths[j] })
	return paths, nil
}

func removeSSHInclude(content, sshDir, configPath string) (string, bool) {
	home, localUser, localHost, err := sshIncludeExpansionContext()
	if err != nil {
		return content, false
	}
	lines := strings.SplitAfter(content, "\n")
	var updated strings.Builder
	found := false
	for _, line := range lines {
		body := strings.TrimSuffix(line, "\n")
		newline := ""
		if strings.HasSuffix(line, "\n") {
			newline = "\n"
		}
		tokens := sshConfigTokens(body)
		directive, arguments := configDirectiveArgs(tokens)
		if directive != "include" || len(arguments) == 0 {
			updated.WriteString(line)
			continue
		}
		remaining := make([]string, 0, len(arguments))
		removedHere := false
		for _, argument := range arguments {
			includedPath, ok := resolveSSHIncludePath(argument.value, sshDir, home, localUser, localHost)
			if ok && includedPath == filepath.Clean(configPath) {
				found = true
				removedHere = true
				continue
			}
			remaining = append(remaining, configValue(argument.value))
		}
		if !removedHere {
			updated.WriteString(line)
			continue
		}
		if len(remaining) == 0 {
			continue
		}
		prefix := body[:tokens[0].start]
		suffix := body[tokens[len(tokens)-1].end:]
		updated.WriteString(prefix)
		updated.WriteString("Include ")
		updated.WriteString(strings.Join(remaining, " "))
		updated.WriteString(suffix)
		updated.WriteString(newline)
	}
	return updated.String(), found
}

func resolveSSHIncludePath(path, sshDir, home, localUser, localHost string) (string, bool) {
	path, err := expandSSHIncludePattern(path, home, localUser, localHost)
	if err != nil || strings.ContainsAny(path, "*?[") {
		return "", false
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(sshDir, path)
	}
	return filepath.Clean(path), true
}

func isWithinSSHDir(path, sshDir string) bool {
	resolvedRoot, err := filepath.EvalSymlinks(sshDir)
	if err != nil {
		return false
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func removeEmptySSHConfigDirs(configPath, sshDir string) (int, error) {
	root, err := filepath.EvalSymlinks(sshDir)
	if err != nil {
		return 0, fmt.Errorf("не удалось определить папку ~/.ssh: %w", err)
	}
	removed := 0
	for directory := filepath.Dir(configPath); filepath.Clean(directory) != filepath.Clean(root); directory = filepath.Dir(directory) {
		relative, err := filepath.Rel(root, directory)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return removed, fmt.Errorf("путь папки вышел за пределы ~/.ssh")
		}
		entries, err := os.ReadDir(directory)
		if err != nil {
			return removed, fmt.Errorf("не удалось проверить папку %s: %w", directory, err)
		}
		if len(entries) != 0 {
			break
		}
		if err := os.Remove(directory); err != nil {
			return removed, fmt.Errorf("не удалось удалить пустую папку %s: %w", directory, err)
		}
		removed++
	}
	return removed, nil
}

func addSSHInclude(content, includePath string) (string, error) {
	for _, line := range strings.Split(content, "\n") {
		tokens := sshConfigTokens(line)
		if len(tokens) == 0 {
			continue
		}
		directive, args := configDirectiveArgs(tokens)
		if directive == "include" {
			for _, arg := range args {
				if filepath.Clean(filepath.FromSlash(arg.value)) == filepath.Clean(filepath.FromSlash(includePath)) {
					return "", fmt.Errorf("Include для %s уже существует", includePath)
				}
			}
		}
	}
	lines := strings.SplitAfter(content, "\n")
	insertAt := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(strings.TrimSuffix(line, "\n"))
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			insertAt += len(line)
			continue
		}
		break
	}
	includeLine := "Include " + configValue(filepath.ToSlash(includePath)) + "\n"
	if insertAt == len(content) && content != "" && !strings.HasSuffix(content, "\n") {
		includeLine = "\n" + includeLine
	}
	return content[:insertAt] + includeLine + content[insertAt:], nil
}
