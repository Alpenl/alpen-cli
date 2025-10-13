package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Loader 负责从磁盘加载并合并配置
type Loader struct {
	baseDir string
}

// NewLoader 构造 Loader
func NewLoader(baseDir string) *Loader {
	return &Loader{baseDir: baseDir}
}

// Load 读取指定路径的配置文件，env 用于加载额外的环境差异文件
func (l *Loader) Load(path string, env string) (*Config, error) {
	fullPath := l.resolvePath(path)
	baseConfig, err := loadSingleConfig(fullPath)
	if err != nil {
		return nil, fmt.Errorf("加载基础配置失败: %w", err)
	}
	if env != "" {
		envPath := l.appendEnvSuffix(fullPath, env)
		if _, err := os.Stat(envPath); err == nil {
			envConfig, err := loadSingleConfig(envPath)
			if err != nil {
				return nil, fmt.Errorf("加载环境配置失败: %w", err)
			}
			mergeConfig(baseConfig, envConfig)
		}
	}
	if err := baseConfig.Validate(); err != nil {
		return nil, err
	}
	return baseConfig, nil
}

func (l *Loader) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if l.baseDir == "" {
		return path
	}
	return filepath.Join(l.baseDir, path)
}

func (l *Loader) appendEnvSuffix(path string, env string) string {
	ext := filepath.Ext(path)
	base := path[:len(path)-len(ext)]
	return fmt.Sprintf("%s.%s%s", base, env, ext)
}

func loadSingleConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Groups == nil {
		cfg.Groups = map[string]Group{}
	}
	for name, group := range cfg.Groups {
		if group.Scripts == nil {
			group.Scripts = map[string]ScriptTemplate{}
		}
		cfg.Groups[name] = group
	}
	if cfg.Menus == nil {
		cfg.Menus = []Menu{}
	} else {
		for i := range cfg.Menus {
			if cfg.Menus[i].Items == nil {
				cfg.Menus[i].Items = []MenuItem{}
			}
		}
	}
	return &cfg, nil
}

func mergeConfig(base *Config, override *Config) {
	for groupName, overrideGroup := range override.Groups {
		group := base.Groups[groupName]
		if group.Scripts == nil {
			group.Scripts = map[string]ScriptTemplate{}
		}
		group.Description = valueOrDefault(overrideGroup.Description, group.Description)
		for scriptName, overrideScript := range overrideGroup.Scripts {
			baseScript := group.Scripts[scriptName]
			merged := baseScript
			if overrideScript.Command != "" {
				merged.Command = overrideScript.Command
			}
			if overrideScript.Description != "" {
				merged.Description = overrideScript.Description
			}
			if overrideScript.Env != nil {
				if merged.Env == nil {
					merged.Env = map[string]string{}
				}
				for k, v := range overrideScript.Env {
					merged.Env[k] = v
				}
			}
			if len(overrideScript.Platforms) > 0 {
				merged.Platforms = overrideScript.Platforms
			}
			group.Scripts[scriptName] = merged
		}
		base.Groups[groupName] = group
	}
	if len(override.Menus) > 0 {
		mergeMenus(base, override.Menus)
	}
}

func valueOrDefault[T comparable](candidate T, fallback T) T {
	var zero T
	if candidate == zero {
		return fallback
	}
	return candidate
}

func mergeMenus(base *Config, overrides []Menu) {
	index := map[string]int{}
	for i, menu := range base.Menus {
		index[menu.Key] = i
	}
	for _, override := range overrides {
		if pos, ok := index[override.Key]; ok {
			existing := base.Menus[pos]
			existing.Title = valueOrDefault(override.Title, existing.Title)
			existing.Description = valueOrDefault(override.Description, existing.Description)
			existing.Group = valueOrDefault(override.Group, existing.Group)
			if len(override.Items) > 0 {
				existing.Items = override.Items
			}
			base.Menus[pos] = existing
		} else {
			base.Menus = append(base.Menus, override)
			index[override.Key] = len(base.Menus) - 1
		}
	}
}
