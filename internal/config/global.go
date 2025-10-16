package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// GlobalConfig 描述 ~/.alpen 下的全局配置
type GlobalConfig struct {
	DefaultConfigPath string   `yaml:"defaultConfigPath"`
	ScriptsRoot       string   `yaml:"scriptsRoot"`
	SearchPaths       []string `yaml:"searchPaths"`
}

const globalConfigFileName = "global.yaml"

// LoadGlobalConfig 读取全局配置，若不存在则返回带默认值的结构体
func LoadGlobalConfig() (*GlobalConfig, error) {
	home, err := ResolveHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, globalConfigFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaultGlobalConfig(home), nil
		}
		return nil, err
	}
	var cfg GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return fillGlobalDefaults(home, &cfg), nil
}

// SaveGlobalConfig 将配置写回 ~/.alpen/global.yaml（默认权限 0644）
func SaveGlobalConfig(cfg *GlobalConfig) error {
	if cfg == nil {
		return fmt.Errorf("global config 不能为空")
	}
	home, err := ResolveHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, globalConfigFileName)
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// EnsureGlobalStructure 创建 ~/.alpen 及基础子目录
func EnsureGlobalStructure() (string, error) {
	home, err := ResolveHomeDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		return "", err
	}
	configDir := filepath.Join(home, "config")
	scriptsDir := filepath.Join(configDir, "scripts")
	testsDir := filepath.Join(scriptsDir, "tests")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		return "", err
	}
	stateDir := filepath.Join(home, stateDirName)
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return "", err
	}
	return home, nil
}

// PrepareConfigPath 确保给定的配置文件存在，若不存在则自动创建模板
func PrepareConfigPath(path string, template []byte) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("配置路径不能为空")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.WriteFile(path, template, 0o644)
}

// GlobalConfigPath 返回 ~/.alpen/global.yaml 路径
func GlobalConfigPath() (string, error) {
	home, err := ResolveHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, globalConfigFileName), nil
}

// DefaultGlobalConfig 返回带默认路径的全局配置副本
func DefaultGlobalConfig() (*GlobalConfig, error) {
	home, err := ResolveHomeDir()
	if err != nil {
		return nil, err
	}
	cfg := defaultGlobalConfig(home)
	return cfg, nil
}

func defaultGlobalConfig(home string) *GlobalConfig {
	configDir := filepath.Join(home, "config")
	return &GlobalConfig{
		DefaultConfigPath: filepath.Join(configDir, "demo.yaml"),
		ScriptsRoot:       filepath.Join(configDir, "scripts"),
		SearchPaths:       []string{configDir},
	}
}

func fillGlobalDefaults(home string, cfg *GlobalConfig) *GlobalConfig {
	if cfg == nil {
		return defaultGlobalConfig(home)
	}
	if strings.TrimSpace(cfg.DefaultConfigPath) == "" {
		cfg.DefaultConfigPath = filepath.Join(home, "config", "demo.yaml")
	}
	if strings.TrimSpace(cfg.ScriptsRoot) == "" {
		cfg.ScriptsRoot = filepath.Join(home, "config", "scripts")
	}
	if len(cfg.SearchPaths) == 0 {
		cfg.SearchPaths = []string{filepath.Dir(cfg.DefaultConfigPath)}
	}
	return cfg
}
