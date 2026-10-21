package sshconfig

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type sshProfileValues struct {
	alias    string
	hostName string
	user     string
	port     string
	identity string
}

func configDirectiveArgs(tokens []configToken) (string, []configToken) {
	if len(tokens) == 0 {
		return "", nil
	}
	directive := tokens[0].value
	arguments := append([]configToken(nil), tokens[1:]...)
	if name, value, found := strings.Cut(directive, "="); found {
		directive = name
		if value != "" {
			arguments = append([]configToken{{value: value}}, arguments...)
		}
	}
	if len(arguments) > 0 && strings.HasPrefix(arguments[0].value, "=") {
		arguments[0].value = strings.TrimPrefix(arguments[0].value, "=")
		if arguments[0].value == "" {
			arguments = arguments[1:]
		}
	}
	return strings.ToLower(directive), arguments
}

func hasSimpleHostBlock(source, expanded, alias string) bool {
	if !hasDirectHostBlock(source, expanded, alias) {
		return false
	}
	start, _, found := hostBlockRange(source, alias)
	if !found {
		return false
	}
	mode := "global"
	var patterns []string
	managed := map[string]struct{}{"hostname": {}, "user": {}, "port": {}, "identityfile": {}}
	offset := 0
	for _, line := range strings.SplitAfter(source, "\n") {
		if offset >= start {
			break
		}
		offset += len(line)
		tokens := sshConfigTokens(strings.TrimSuffix(line, "\n"))
		if len(tokens) == 0 {
			continue
		}
		directive, arguments := configDirectiveArgs(tokens)
		switch directive {
		case "host":
			mode = "host"
			patterns = patterns[:0]
			for _, token := range arguments {
				patterns = append(patterns, token.value)
			}
			continue
		case "match":
			mode = "match"
			continue
		case "include":
			return false
		}
		if _, isManaged := managed[directive]; !isManaged {
			continue
		}
		switch mode {
		case "global", "match":
			return false
		case "host":
			if hostPatternsMatch(patterns, alias) {
				return false
			}
		}
	}
	return true
}

func hasDirectHostBlock(source, expanded, alias string) bool {
	if _, _, found := hostBlockRange(source, alias); !found {
		return false
	}
	declarations := 0
	for _, line := range strings.Split(expanded, "\n") {
		tokens := sshConfigTokens(line)
		directive, arguments := configDirectiveArgs(tokens)
		if directive != "host" {
			continue
		}
		for _, pattern := range arguments {
			if pattern.value == alias {
				declarations++
			}
		}
	}
	return declarations == 1
}

func hostBlockRange(content, alias string) (int, int, bool) {
	lines := strings.SplitAfter(content, "\n")
	offset := 0
	start, end, matches := -1, len(content), 0
	active := false
	for _, line := range lines {
		lineStart := offset
		offset += len(line)
		tokens := sshConfigTokens(strings.TrimSuffix(line, "\n"))
		if len(tokens) == 0 {
			continue
		}
		directive, arguments := configDirectiveArgs(tokens)
		if directive == "host" || directive == "match" {
			if active {
				end = lineStart
				active = false
			}
			if directive == "host" && len(arguments) == 1 && arguments[0].value == alias {
				matches++
				start = lineStart
				active = true
			}
		}
	}
	if matches != 1 {
		return 0, 0, false
	}
	return start, end, true
}

func validateSSHProfile(values sshProfileValues) error {
	if values.alias == "" || len(strings.Fields(values.alias)) != 1 || strings.ContainsAny(values.alias, "*!?#\r\n") {
		return fmt.Errorf("Host должен быть одним конкретным алиасом без пробелов и шаблонов")
	}
	for directive, value := range map[string]string{
		"HostName":     values.hostName,
		"User":         values.user,
		"Port":         values.port,
		"IdentityFile": values.identity,
	} {
		if strings.ContainsAny(value, "\r\n\x00") {
			return fmt.Errorf("%s не может содержать перенос строки", directive)
		}
	}
	if values.port != "" {
		port, err := strconv.Atoi(values.port)
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("Port должен быть числом от 1 до 65535")
		}
	}
	return nil
}

func createHostBlock(content string, values sshProfileValues) string {
	var block strings.Builder
	block.WriteString("Host ")
	block.WriteString(configValue(values.alias))
	block.WriteByte('\n')
	writeConfigOption(&block, "HostName", values.hostName)
	writeConfigOption(&block, "User", values.user)
	writeConfigOption(&block, "Port", values.port)
	writeConfigOption(&block, "IdentityFile", values.identity)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	if content != "" && !strings.HasSuffix(content, "\n\n") {
		content += "\n"
	}
	return content + block.String()
}

func updateHostBlock(content, existingAlias string, values sshProfileValues) (string, error) {
	start, end, found := hostBlockRange(content, existingAlias)
	if !found {
		return "", fmt.Errorf("не найдена простая запись Host %s в ~/.ssh/config", existingAlias)
	}
	block := strings.SplitAfter(content[start:end], "\n")
	if len(block) == 0 {
		return "", fmt.Errorf("не удалось прочитать запись Host %s", values.alias)
	}
	var updated strings.Builder
	updated.WriteString("Host ")
	updated.WriteString(configValue(values.alias))
	updated.WriteByte('\n')
	writeConfigOption(&updated, "HostName", values.hostName)
	writeConfigOption(&updated, "User", values.user)
	writeConfigOption(&updated, "Port", values.port)
	writeConfigOption(&updated, "IdentityFile", values.identity)
	managed := map[string]struct{}{"hostname": {}, "user": {}, "port": {}, "identityfile": {}}
	for _, line := range block[1:] {
		tokens := sshConfigTokens(strings.TrimSuffix(line, "\n"))
		if len(tokens) > 0 {
			directive, _ := configDirectiveArgs(tokens)
			if _, ok := managed[directive]; ok {
				continue
			}
		}
		updated.WriteString(line)
	}
	return content[:start] + updated.String() + content[end:], nil
}

func renameHostBlock(content, existingAlias, alias string) (string, error) {
	start, end, found := hostBlockRange(content, existingAlias)
	if !found {
		return "", fmt.Errorf("не найдена простая запись Host %s в ~/.ssh/config", existingAlias)
	}
	block := content[start:end]
	lineEnd := strings.IndexByte(block, '\n')
	if lineEnd < 0 {
		lineEnd = len(block)
	}
	remainder := ""
	if lineEnd < len(block) {
		remainder = block[lineEnd+1:]
	}
	updated := "Host " + configValue(alias) + "\n" + remainder
	return content[:start] + updated + content[end:], nil
}

func deleteHostBlock(content, alias string) (string, error) {
	start, end, found := hostBlockRange(content, alias)
	if !found {
		return "", fmt.Errorf("не найдена простая запись Host %s в ~/.ssh/config", alias)
	}
	return content[:start] + content[end:], nil
}

func writeConfigOption(builder *strings.Builder, directive, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	builder.WriteString("    ")
	builder.WriteString(directive)
	builder.WriteByte(' ')
	builder.WriteString(configValue(value))
	builder.WriteByte('\n')
}

func configValue(value string) string {
	if !strings.ContainsAny(value, " \t#\"\\") {
		return value
	}
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return `"` + value + `"`
}

func readSSHConfigForEdit(path string) (string, os.FileMode, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", 0o600, nil
		}
		return "", 0, fmt.Errorf("не удалось проверить SSH config: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", 0, fmt.Errorf("SSH config не является обычным файлом")
	}
	content, err := readTextFile(path)
	if err != nil {
		return "", 0, err
	}
	return content, info.Mode().Perm(), nil
}

func writeSSHConfig(path, content string, mode os.FileMode) error {
	if mode == 0 {
		mode = 0o600
	}
	if err := validateSSHConfig(path, []byte(content)); err != nil {
		return fmt.Errorf("ошибка синтаксиса SSH config: %w", err)
	}
	if err := replaceFile(path, []byte(content), mode); err != nil {
		return fmt.Errorf("не удалось сохранить SSH config: %w", err)
	}
	return nil
}
