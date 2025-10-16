package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultHomeDirName    = ".alpen"
	stateDirName          = "state"
	activeConfigFileName  = "active-config"
	envHomeOverride       = "ALPEN_HOME"
	defaultDirPermission  = 0o700
	defaultFilePermission = 0o644
)

// ResolveHomeDir 获取 Alpen CLI 全局配置目录（可通过 ALPEN_HOME 覆盖）
func ResolveHomeDir() (string, error) {
	if custom := strings.TrimSpace(os.Getenv(envHomeOverride)); custom != "" {
		return custom, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, defaultHomeDirName), nil
}

// LoadActiveConfigPath 读取当前激活的配置文件路径
func LoadActiveConfigPath() (string, error) {
	home, err := ResolveHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, stateDirName, activeConfigFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// SaveActiveConfigPath 将选中的配置文件路径写入全局状态
func SaveActiveConfigPath(configPath string) error {
	home, err := ResolveHomeDir()
	if err != nil {
		return err
	}
	stateDir := filepath.Join(home, stateDirName)
	if err := os.MkdirAll(stateDir, defaultDirPermission); err != nil {
		return err
	}
	path := filepath.Join(stateDir, activeConfigFileName)
	return os.WriteFile(path, []byte(strings.TrimSpace(configPath)), defaultFilePermission)
}

// LoadProjectActiveConfigPath 读取项目级激活配置路径（优先级高于全局）
func LoadProjectActiveConfigPath(projectConfigDir string) (string, error) {
	dir := strings.TrimSpace(projectConfigDir)
	if dir == "" {
		return "", nil
	}
	path := filepath.Join(dir, stateDirName, activeConfigFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// SaveProjectActiveConfigPath 写入项目级激活配置路径
func SaveProjectActiveConfigPath(projectConfigDir string, configPath string) error {
	dir := strings.TrimSpace(projectConfigDir)
	if dir == "" {
		return errors.New("project config directory is empty")
	}
	stateDir := filepath.Join(dir, stateDirName)
	if err := os.MkdirAll(stateDir, defaultDirPermission); err != nil {
		return err
	}
	path := filepath.Join(stateDir, activeConfigFileName)
	return os.WriteFile(path, []byte(strings.TrimSpace(configPath)), defaultFilePermission)
}
