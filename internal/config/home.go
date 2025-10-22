package config

import (
	"os"
	"path/filepath"
)

const (
	configDirName  = "config"
	scriptsDirName = "scripts"
	testsDirName   = "tests"
)

// EnsureHomeStructure 会创建 ~/.alpen 及基础子目录，返回绝对路径
func EnsureHomeStructure() (string, error) {
	home, err := ResolveHomeDir()
	if err != nil {
		return "", err
	}
	if err := mkdirAll(home, defaultDirPermission); err != nil {
		return "", err
	}
	configDir := filepath.Join(home, configDirName)
	if err := mkdirAll(configDir, 0o755); err != nil {
		return "", err
	}
	scriptsDir := filepath.Join(configDir, scriptsDirName)
	if err := mkdirAll(scriptsDir, 0o755); err != nil {
		return "", err
	}
	testsDir := filepath.Join(scriptsDir, testsDirName)
	if err := mkdirAll(testsDir, 0o755); err != nil {
		return "", err
	}
	stateDir := filepath.Join(home, stateDirName)
	if err := mkdirAll(stateDir, defaultDirPermission); err != nil {
		return "", err
	}
	return home, nil
}

// DefaultConfigPath 返回 ~/.alpen/config/demo.yaml 的绝对路径
func DefaultConfigPath() (string, error) {
	return NormalizeConfigPath("")
}

// DefaultScriptsRoot 返回 ~/.alpen/config/scripts 的绝对路径
func DefaultScriptsRoot() (string, error) {
	return NormalizeScriptsRoot("")
}

// ConfigDir 返回 ~/.alpen/config 的绝对路径
func ConfigDir() (string, error) {
	home, err := ResolveHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, configDirName), nil
}

func mkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}
