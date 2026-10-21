package sshconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Alias contains the SSH options shown for one concrete Host pattern.
type Alias struct {
	Name          string
	HostName      string
	User          string
	Port          string
	Identity      string
	HostEditable  bool
	IdentityCount int
	Editable      bool
}

// Config contains aliases found in a config file and whether it can be deleted.
type Config struct {
	Path      string
	Aliases   []Alias
	Deletable bool
}

// Snapshot is the loaded SSH config state and any non-fatal warnings.
type Snapshot struct {
	Configs []Config
	Warning string
}

// AliasValues holds editable options for a Host block.
type AliasValues struct {
	Name     string
	HostName string
	User     string
	Port     string
	Identity string
}

// DeleteResult describes the effects of deleting an included config.
type DeleteResult struct {
	Deleted        bool
	RemovedInclude bool
	RemovedDirs    int
}

// Service reads and updates the user's SSH configuration under sshDir.
type Service struct {
	sshDir     string
	mainConfig string
}

// NewService creates a config service rooted at the user's .ssh directory.
func NewService(sshDir string) *Service {
	return &Service{sshDir: sshDir, mainConfig: filepath.Join(sshDir, "config")}
}

// MainConfigPath returns the path to the primary SSH config.
func (s *Service) MainConfigPath() string {
	return s.mainConfig
}

// Load reads the primary config, its directly included files, and their aliases.
func (s *Service) Load() (Snapshot, error) {
	mainContent, _, err := readSSHConfigForEdit(s.mainConfig)
	if err != nil {
		return Snapshot{}, err
	}

	effectiveConfig, validationErr := prepareSSHConfigForValidation(s.mainConfig, mainContent)
	var warnings []string
	if validationErr != nil {
		warnings = append(warnings, "SSH config требует исправления: "+validationErr.Error())
		effectiveConfig = ""
	}

	includePaths, includeErr := sshConfigIncludes(mainContent, s.sshDir)
	if includeErr != nil {
		warnings = append(warnings, "Не удалось разобрать Include: "+includeErr.Error())
	}
	configs := []Config{{Path: s.mainConfig}}
	for _, includePath := range includePaths {
		if includePath == s.mainConfig {
			continue
		}
		if _, err := os.Stat(includePath); err != nil {
			continue
		}
		configs = append(configs, Config{
			Path:      includePath,
			Deletable: isWithinSSHDir(includePath, s.sshDir),
		})
	}

	for configIndex := range configs {
		config := &configs[configIndex]
		source, _, readErr := readSSHConfigForEdit(config.Path)
		if readErr != nil {
			warnings = append(warnings, readErr.Error())
			continue
		}
		config.Aliases = inspectSSHAliases(source)
		for aliasIndex := range config.Aliases {
			alias := &config.Aliases[aliasIndex]
			alias.HostEditable = effectiveConfig != "" && hasDirectHostBlock(source, effectiveConfig, alias.Name)
			alias.Editable = alias.HostEditable && hasSimpleHostBlock(source, effectiveConfig, alias.Name) && alias.IdentityCount <= 1
		}
	}

	return Snapshot{Configs: configs, Warning: strings.Join(warnings, "; ")}, nil
}

// CreateConfig creates a config below sshDir and adds its Include to the main file.
func (s *Service) CreateConfig(relativePath string) (string, error) {
	relativePath = strings.TrimSpace(relativePath)
	if relativePath == "" || strings.HasPrefix(relativePath, "~") {
		return "", fmt.Errorf("укажите путь внутри ~/.ssh")
	}
	clean := filepath.Clean(filepath.FromSlash(strings.TrimLeft(relativePath, "/")))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("путь конфига должен оставаться внутри ~/.ssh")
	}
	root, err := filepath.EvalSymlinks(s.sshDir)
	if err != nil {
		return "", fmt.Errorf("не удалось определить папку ~/.ssh: %w", err)
	}
	parts := strings.Split(clean, string(filepath.Separator))
	currentDir := root
	for _, part := range parts[:len(parts)-1] {
		currentDir = filepath.Join(currentDir, part)
		if err := os.Mkdir(currentDir, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("не удалось создать папку: %w", err)
		}
		info, err := os.Lstat(currentDir)
		if err != nil {
			return "", fmt.Errorf("не удалось проверить папку: %w", err)
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("путь содержит компонент, который не является обычной папкой")
		}
	}
	configPath := filepath.Join(s.sshDir, clean)
	if _, err := os.Lstat(configPath); err == nil {
		return "", fmt.Errorf("файл уже существует: %s", configPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	mainContent, mainMode, err := readSSHConfigForEdit(s.mainConfig)
	if err != nil {
		return "", err
	}
	includePath, err := filepath.Rel(s.sshDir, configPath)
	if err != nil {
		return "", err
	}
	updatedMain, err := addSSHInclude(mainContent, filepath.ToSlash(includePath))
	if err != nil {
		return "", err
	}
	if err := writeSSHConfig(configPath, "", 0o600); err != nil {
		return "", err
	}
	if err := writeSSHConfig(s.mainConfig, updatedMain, mainMode); err != nil {
		_ = os.Remove(configPath)
		return "", err
	}
	return configPath, nil
}

// DeleteConfig removes an included file, its Include entry, and empty parent directories.
func (s *Service) DeleteConfig(configPath string) (DeleteResult, error) {
	if filepath.Clean(configPath) == filepath.Clean(s.mainConfig) || !isWithinSSHDir(configPath, s.sshDir) {
		return DeleteResult{}, fmt.Errorf("этот конфиг нельзя удалить")
	}
	mainContent, mainMode, err := readSSHConfigForEdit(s.mainConfig)
	if err != nil {
		return DeleteResult{}, err
	}
	updatedMain, found := removeSSHInclude(mainContent, s.sshDir, configPath)
	if !found {
		includedPaths, includeErr := sshConfigIncludes(mainContent, s.sshDir)
		if includeErr != nil {
			return DeleteResult{}, fmt.Errorf("не удалось разобрать Include: %w", includeErr)
		}
		included := false
		for _, includedPath := range includedPaths {
			if filepath.Clean(includedPath) == filepath.Clean(configPath) {
				included = true
				break
			}
		}
		if !included {
			return DeleteResult{}, fmt.Errorf("этот файл больше не подключён в основном config")
		}
		updatedMain = mainContent
	}
	if _, _, err := readSSHConfigForEdit(configPath); err != nil {
		return DeleteResult{}, err
	}
	resolvedPath, err := filepath.EvalSymlinks(configPath)
	if err != nil {
		return DeleteResult{}, fmt.Errorf("не удалось проверить путь конфига: %w", err)
	}
	if err := writeSSHConfig(s.mainConfig, updatedMain, mainMode); err != nil {
		return DeleteResult{}, err
	}
	if err := os.Remove(configPath); err != nil {
		rollbackErr := writeSSHConfig(s.mainConfig, mainContent, mainMode)
		if rollbackErr != nil {
			return DeleteResult{}, fmt.Errorf("не удалось удалить конфиг (%v); не удалось восстановить Include (%v)", err, rollbackErr)
		}
		return DeleteResult{}, fmt.Errorf("не удалось удалить конфиг: %w", err)
	}
	removedDirs, cleanupErr := removeEmptySSHConfigDirs(resolvedPath, s.sshDir)
	if cleanupErr != nil {
		return DeleteResult{Deleted: true, RemovedInclude: found, RemovedDirs: removedDirs}, fmt.Errorf("конфиг удалён, но не удалось очистить пустые папки: %w", cleanupErr)
	}
	return DeleteResult{Deleted: true, RemovedInclude: found, RemovedDirs: removedDirs}, nil
}

// SaveAlias creates a Host block or updates an existing editable block.
func (s *Service) SaveAlias(configPath, existingAlias string, values AliasValues, creating bool) error {
	profile := sshProfileValues{
		alias:    strings.TrimSpace(values.Name),
		hostName: strings.TrimSpace(values.HostName),
		user:     strings.TrimSpace(values.User),
		port:     strings.TrimSpace(values.Port),
		identity: strings.TrimSpace(values.Identity),
	}
	if err := validateSSHProfile(profile); err != nil {
		return err
	}
	content, mode, err := readSSHConfigForEdit(configPath)
	if err != nil {
		return err
	}
	aliases := inspectSSHAliases(content)
	if creating {
		for _, alias := range aliases {
			if strings.EqualFold(alias.Name, profile.alias) {
				return fmt.Errorf("такой Host уже есть в SSH config")
			}
		}
		return writeSSHConfig(configPath, createHostBlock(content, profile), mode)
	}
	if profile.alias != existingAlias {
		for _, alias := range aliases {
			if alias.Name != existingAlias && strings.EqualFold(alias.Name, profile.alias) {
				return fmt.Errorf("такой Host уже есть в SSH config")
			}
		}
	}
	mainContent, _, err := readSSHConfigForEdit(s.mainConfig)
	if err != nil {
		return err
	}
	effectiveConfig, err := prepareSSHConfigForValidation(s.mainConfig, mainContent)
	if err != nil || !hasDirectHostBlock(content, effectiveConfig, existingAlias) {
		return fmt.Errorf("Host нельзя безопасно изменить в этом SSH config")
	}
	if hasSimpleHostBlock(content, effectiveConfig, existingAlias) {
		updated, err := updateHostBlock(content, existingAlias, profile)
		if err != nil {
			return err
		}
		return writeSSHConfig(configPath, updated, mode)
	}
	if profile.alias == existingAlias {
		return fmt.Errorf("эти параметры нельзя безопасно изменить в форме")
	}
	updated, err := renameHostBlock(content, existingAlias, profile.alias)
	if err != nil {
		return err
	}
	return writeSSHConfig(configPath, updated, mode)
}

// DeleteAlias removes a concrete Host block from the selected config file.
func (s *Service) DeleteAlias(configPath, aliasName string) error {
	content, mode, err := readSSHConfigForEdit(configPath)
	if err != nil {
		return err
	}
	updated, err := deleteHostBlock(content, aliasName)
	if err != nil {
		return err
	}
	return writeSSHConfig(configPath, updated, mode)
}
