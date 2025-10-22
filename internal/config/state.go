package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultHomeDirName    = ".alpen"
	stateDirName          = "state"
	activeConfigFileName  = "active-config"
	defaultDirPermission  = 0o700
	defaultFilePermission = 0o644
)

// ResolveHomeDir 获取 Alpen CLI 的用户目录（固定为用户目录下的 .alpen）
func ResolveHomeDir() (string, error) {
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

// SaveActiveConfigPath 将选中的配置文件路径写入状态目录
func SaveActiveConfigPath(configPath string) error {
	home, err := ResolveHomeDir()
	if err != nil {
		return err
	}
	normalized, err := NormalizeConfigPath(configPath)
	if err != nil {
		return err
	}
	stateDir := filepath.Join(home, stateDirName)
	if err := os.MkdirAll(stateDir, defaultDirPermission); err != nil {
		return err
	}
	path := filepath.Join(stateDir, activeConfigFileName)
	return os.WriteFile(path, []byte(normalized), defaultFilePermission)
}
