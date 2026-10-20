package ui

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

const maxViewFileSize = 2 << 20
const sshConfigValidationTimeout = 5 * time.Second
const maxValidationConfigSize = 8 << 20
const maxIncludeDepth = 16

func readTextFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("не удалось прочитать %s: %w", path, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("не удалось проверить %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s не является обычным файлом", path)
	}
	if info.Size() > maxViewFileSize {
		return "", fmt.Errorf("файл %s больше 2 МиБ и не открыт", path)
	}

	data, err := io.ReadAll(io.LimitReader(file, int64(maxViewFileSize)+1))
	if err != nil {
		return "", fmt.Errorf("не удалось прочитать %s: %w", path, err)
	}
	if len(data) > maxViewFileSize {
		return "", fmt.Errorf("файл %s больше 2 МиБ и не открыт", path)
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("%s не похож на текстовый файл", path)
	}
	return string(data), nil
}

func validateSSHConfig(path string, content []byte) error {
	tempFile, err := os.CreateTemp(filepath.Dir(path), ".mssh-config-check-*")
	if err != nil {
		return fmt.Errorf("не удалось создать временный файл проверки: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	validationContent, err := prepareSSHConfigForValidation(path, string(content))
	if err != nil {
		tempFile.Close()
		return fmt.Errorf("не удалось подготовить SSH config к проверке: %w", err)
	}
	if _, err := tempFile.WriteString(validationContent); err != nil {
		tempFile.Close()
		return fmt.Errorf("не удалось записать временный файл проверки: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("не удалось закрыть временный файл проверки: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), sshConfigValidationTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, "ssh", "-G", "-F", tempPath, "mssh-config-validation.invalid")
	output, err := command.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("проверка SSH config превысила лимит времени %s", sshConfigValidationTimeout)
		}
		if message := strings.TrimSpace(string(output)); message != "" {
			return fmt.Errorf("%s", message)
		}
		return fmt.Errorf("не удалось запустить ssh -G: %w", err)
	}
	return nil
}

type configToken struct {
	value string
	start int
	end   int
}

func prepareSSHConfigForValidation(path, content string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("не удалось определить домашнюю папку: %w", err)
	}
	currentUser, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("не удалось определить пользователя: %w", err)
	}
	localHost, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("не удалось определить имя компьютера: %w", err)
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("не удалось разрешить путь конфига: %w", err)
	}
	activeIncludes := map[string]struct{}{path: {}}
	includeRoot := filepath.Join(home, ".ssh")
	content, totalSize, err := expandSSHConfigIncludes(content, includeRoot, home, currentUser.Username, localHost, activeIncludes, 0, len(content))
	if err != nil {
		return "", err
	}
	if totalSize > maxValidationConfigSize {
		return "", fmt.Errorf("размер конфига с Include превышает %d МиБ", maxValidationConfigSize>>20)
	}
	if err := validateSSHConfigQuotes(content); err != nil {
		return "", err
	}
	return sanitizeMatchExec(content), nil
}

func expandSSHConfigIncludes(content, includeRoot, home, localUser, localHost string, active map[string]struct{}, depth, totalSize int) (string, int, error) {
	if depth > maxIncludeDepth {
		return "", totalSize, fmt.Errorf("превышена максимальная глубина Include (%d)", maxIncludeDepth)
	}
	lines := strings.SplitAfter(content, "\n")
	var expanded strings.Builder
	for _, line := range lines {
		body := strings.TrimSuffix(line, "\n")
		tokens := sshConfigTokens(body)
		if len(tokens) == 0 || !strings.EqualFold(tokens[0].value, "Include") || len(tokens) == 1 {
			expanded.WriteString(line)
			continue
		}

		for _, token := range tokens[1:] {
			includePattern, err := expandSSHIncludePattern(token.value, home, localUser, localHost)
			if err != nil {
				return "", totalSize, err
			}
			if !filepath.IsAbs(includePattern) {
				includePattern = filepath.Join(includeRoot, includePattern)
			}
			matches, err := filepath.Glob(includePattern)
			if err != nil {
				return "", totalSize, fmt.Errorf("неверный шаблон Include %q: %w", token.value, err)
			}
			for _, includePath := range matches {
				resolvedPath, err := filepath.EvalSymlinks(includePath)
				if err != nil {
					return "", totalSize, fmt.Errorf("не удалось открыть Include %s: %w", includePath, err)
				}
				if _, exists := active[resolvedPath]; exists {
					return "", totalSize, fmt.Errorf("обнаружен цикл Include: %s", resolvedPath)
				}
				included, err := readValidationInclude(resolvedPath)
				if err != nil {
					return "", totalSize, err
				}
				totalSize += len(included)
				if totalSize > maxValidationConfigSize {
					return "", totalSize, fmt.Errorf("размер конфига с Include превышает %d МиБ", maxValidationConfigSize>>20)
				}
				active[resolvedPath] = struct{}{}
				expandedInclude, includedSize, err := expandSSHConfigIncludes(included, includeRoot, home, localUser, localHost, active, depth+1, totalSize)
				delete(active, resolvedPath)
				if err != nil {
					return "", includedSize, err
				}
				totalSize = includedSize
				expanded.WriteString(expandedInclude)
				if !strings.HasSuffix(expandedInclude, "\n") {
					expanded.WriteByte('\n')
				}
			}
		}
	}
	return expanded.String(), totalSize, nil
}

func expandSSHIncludePattern(pattern, home, localUser, localHost string) (string, error) {
	pattern = os.ExpandEnv(pattern)
	if pattern == "~" {
		pattern = home
	} else if strings.HasPrefix(pattern, "~/") {
		pattern = filepath.Join(home, pattern[2:])
	} else if strings.HasPrefix(pattern, "~") {
		username, relativePath, found := strings.Cut(pattern[1:], string(filepath.Separator))
		if !found {
			relativePath = ""
		}
		includedUser, err := user.Lookup(username)
		if err != nil {
			return "", fmt.Errorf("не удалось определить домашнюю папку пользователя %s: %w", username, err)
		}
		pattern = filepath.Join(includedUser.HomeDir, relativePath)
	}

	var expanded strings.Builder
	for index := 0; index < len(pattern); index++ {
		if pattern[index] != '%' || index+1 >= len(pattern) {
			expanded.WriteByte(pattern[index])
			continue
		}
		index++
		switch pattern[index] {
		case '%':
			expanded.WriteByte('%')
		case 'd':
			expanded.WriteString(home)
		case 'h', 'n':
			expanded.WriteString("mssh-config-validation.invalid")
		case 'p':
			expanded.WriteString("22")
		case 'r', 'u':
			expanded.WriteString(localUser)
		case 'L':
			expanded.WriteString(strings.SplitN(localHost, ".", 2)[0])
		case 'l':
			expanded.WriteString(localHost)
		default:
			return "", fmt.Errorf("неподдерживаемый токен %%%c в Include", pattern[index])
		}
	}
	return expanded.String(), nil
}

func readValidationInclude(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("не удалось открыть Include %s: %w", path, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("не удалось проверить Include %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("Include %s не является обычным файлом", path)
	}
	if info.Size() > maxViewFileSize {
		return "", fmt.Errorf("Include %s больше 2 МиБ", path)
	}
	content, err := io.ReadAll(io.LimitReader(file, int64(maxViewFileSize)+1))
	if err != nil {
		return "", fmt.Errorf("не удалось прочитать Include %s: %w", path, err)
	}
	if len(content) > maxViewFileSize {
		return "", fmt.Errorf("Include %s больше 2 МиБ", path)
	}
	return string(content), nil
}

func sanitizeMatchExec(content string) string {
	lines := strings.SplitAfter(content, "\n")
	for index, line := range lines {
		body := strings.TrimSuffix(line, "\n")
		tokens := sshConfigTokens(body)
		if len(tokens) == 0 {
			continue
		}

		if !strings.EqualFold(tokens[0].value, "Match") {
			continue
		}

		var execArguments []configToken
		for tokenIndex := 1; tokenIndex < len(tokens); {
			criterion := strings.ToLower(tokens[tokenIndex].value)
			switch criterion {
			case "all", "canonical", "final":
				tokenIndex++
				continue
			}
			if tokenIndex+1 >= len(tokens) {
				break
			}
			if criterion == "exec" {
				execArguments = append(execArguments, tokens[tokenIndex+1])
			}
			tokenIndex += 2
		}
		for argumentIndex := len(execArguments) - 1; argumentIndex >= 0; argumentIndex-- {
			argument := execArguments[argumentIndex]
			body = body[:argument.start] + `":"` + body[argument.end:]
		}
		newline := ""
		if strings.HasSuffix(line, "\n") {
			newline = "\n"
		}
		lines[index] = body + newline
	}
	return strings.Join(lines, "")
}

func sshConfigTokens(line string) []configToken {
	var tokens []configToken
	for index := 0; index < len(line); {
		for index < len(line) && (line[index] == ' ' || line[index] == '\t' || line[index] == '\r') {
			index++
		}
		if index >= len(line) || line[index] == '#' {
			break
		}

		start := index
		var value strings.Builder
		inQuotes := false
		for index < len(line) {
			character := line[index]
			if character == '"' {
				inQuotes = !inQuotes
				index++
				continue
			}
			if character == '\\' && inQuotes && index+1 < len(line) {
				index++
				value.WriteByte(line[index])
				index++
				continue
			}
			if !inQuotes && (character == ' ' || character == '\t' || character == '\r' || character == '#') {
				break
			}
			value.WriteByte(character)
			index++
		}
		tokens = append(tokens, configToken{value: value.String(), start: start, end: index})
		if index < len(line) && line[index] == '#' {
			break
		}
	}
	return tokens
}

func validateSSHConfigQuotes(content string) error {
	for lineNumber, line := range strings.Split(content, "\n") {
		inQuotes := false
		for index := 0; index < len(line); index++ {
			if line[index] == '\\' && inQuotes && index+1 < len(line) {
				index++
				continue
			}
			if line[index] == '"' {
				inQuotes = !inQuotes
				continue
			}
			if !inQuotes && line[index] == '#' {
				break
			}
		}
		if inQuotes {
			return fmt.Errorf("незакрытая кавычка в строке %d", lineNumber+1)
		}
	}
	return nil
}

func replaceFile(path string, content []byte, mode os.FileMode) error {
	tempFile, err := os.CreateTemp(filepath.Dir(path), ".mssh-config-save-*")
	if err != nil {
		return fmt.Errorf("не удалось создать временный файл: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if _, err := tempFile.Write(content); err != nil {
		tempFile.Close()
		return fmt.Errorf("не удалось записать временный файл: %w", err)
	}
	if err := tempFile.Chmod(mode.Perm()); err != nil {
		tempFile.Close()
		return fmt.Errorf("не удалось установить права файла: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return fmt.Errorf("не удалось синхронизировать файл: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("не удалось закрыть временный файл: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("не удалось заменить файл: %w", err)
	}
	return nil
}
