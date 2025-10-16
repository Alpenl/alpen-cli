package config

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Loader 负责从磁盘加载并合并配置
type Loader struct {
	baseDir     string
	diagnostics []Diagnostic
}

// NewLoader 构造 Loader
func NewLoader(baseDir string) *Loader {
	return &Loader{baseDir: baseDir}
}

// Diagnostic 用于记录配置合并过程中的提示
type Diagnostic struct {
	Level   string
	Message string
}

// Diagnostics 返回最近一次 Load 产生的诊断信息
func (l *Loader) Diagnostics() []Diagnostic {
	result := make([]Diagnostic, len(l.diagnostics))
	copy(result, l.diagnostics)
	return result
}

// SourceInfo 描述命令或动作的来源信息
type SourceInfo struct {
	Module string
	File   string
}

func (s SourceInfo) String() string {
	switch {
	case strings.TrimSpace(s.Module) != "" && strings.TrimSpace(s.File) != "":
		return fmt.Sprintf("%s (%s)", s.Module, s.File)
	case strings.TrimSpace(s.File) != "":
		return s.File
	case strings.TrimSpace(s.Module) != "":
		return s.Module
	default:
		return ""
	}
}

// Load 读取指定路径的配置文件，env 用于加载额外的环境差异文件
func (l *Loader) Load(path string, env string) (*Config, error) {
	l.diagnostics = nil
	fullPath := l.resolvePath(path)
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("加载基础配置失败: %w", err)
	}
	if info.IsDir() {
		dirCfg, err := l.loadDirectoryConfig(fullPath)
		if err != nil {
			return nil, err
		}
		dirCfg.Diagnostics = l.Diagnostics()
		if err := dirCfg.Validate(); err != nil {
			return nil, err
		}
		return dirCfg, nil
	}

	baseConfig, err := loadSingleConfig(fullPath, l.describeSource(fullPath, ""))
	if err != nil {
		return nil, fmt.Errorf("加载基础配置失败: %w", err)
	}
	if env != "" {
		envPath := l.appendEnvSuffix(fullPath, env)
		if _, err := os.Stat(envPath); err == nil {
			envConfig, err := loadSingleConfig(envPath, l.describeSource(envPath, fmt.Sprintf("@env:%s", env)))
			if err != nil {
				return nil, fmt.Errorf("加载环境配置失败: %w", err)
			}
			mergeConfig(baseConfig, envConfig, mergeOptions{})
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

func loadSingleConfig(path string, source SourceInfo) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	normalizeConfig(&cfg)
	registerOrigins(&cfg, source)
	return &cfg, nil
}

func normalizeConfig(cfg *Config) {
	if cfg.Commands == nil {
		cfg.Commands = map[string]CommandSpec{}
		return
	}
	for name, spec := range cfg.Commands {
		if spec.Actions == nil {
			spec.Actions = map[string]ActionSpec{}
		}
		cfg.Commands[name] = spec
	}
}

type mergeOptions struct {
	trackOverrides bool
	label          string
	collect        func(Diagnostic)
}

func mergeConfig(base *Config, override *Config, opts mergeOptions) {
	if base.Commands == nil {
		base.Commands = map[string]CommandSpec{}
	}
	for name, overrideSpec := range override.Commands {
		existingSpec, exists := base.Commands[name]
		if !exists {
			base.Commands[name] = overrideSpec
			continue
		}
		baseSpec := existingSpec
		if baseSpec.Actions == nil {
			baseSpec.Actions = map[string]ActionSpec{}
		}
		existed := baseSpec.Command != "" || len(baseSpec.Actions) > 0 || baseSpec.Alias != "" || baseSpec.Description != ""
		overrideDetected := false
		if overrideSpec.Alias != "" && overrideSpec.Alias != baseSpec.Alias {
			if baseSpec.Alias != "" {
				overrideDetected = true
			}
			baseSpec.Alias = overrideSpec.Alias
		}
		if overrideSpec.Description != "" && overrideSpec.Description != baseSpec.Description {
			if baseSpec.Description != "" {
				overrideDetected = true
			}
			baseSpec.Description = overrideSpec.Description
		}
		if overrideSpec.Command != "" && overrideSpec.Command != baseSpec.Command {
			if baseSpec.Command != "" {
				overrideDetected = true
			}
			baseSpec.Command = overrideSpec.Command
		}
		for actionName, overrideAction := range overrideSpec.Actions {
			baseAction := baseSpec.Actions[actionName]
			actionExisted := baseAction.Command != "" || baseAction.Description != "" || baseAction.Alias != ""
			previousOrigin := baseAction.Origin
			if overrideAction.Alias != "" && overrideAction.Alias != baseAction.Alias {
				baseAction.Alias = overrideAction.Alias
			}
			if overrideAction.Description != "" && overrideAction.Description != baseAction.Description {
				baseAction.Description = overrideAction.Description
			}
			if overrideAction.Command != "" && overrideAction.Command != baseAction.Command {
				baseAction.Command = overrideAction.Command
			}
			baseAction.Origin = overrideAction.Origin
			baseSpec.Actions[actionName] = baseAction
			if opts.trackOverrides && actionExisted && opts.collect != nil {
				opts.collect(Diagnostic{
					Level:   "warning",
					Message: fmt.Sprintf("子命令 %s.%s 被 %s 覆盖，原来源: %s，新来源: %s", name, actionName, opts.label, previousOrigin.String(), overrideAction.Origin.String()),
				})
			}
			if actionExisted {
				overrideDetected = true
			}
		}
		if opts.trackOverrides && existed && overrideDetected && opts.collect != nil {
			opts.collect(Diagnostic{
				Level:   "warning",
				Message: fmt.Sprintf("命令 %s 被 %s 覆盖，原来源: %s，新来源: %s", name, opts.label, baseSpec.Origin.String(), overrideSpec.Origin.String()),
			})
		}
		if overrideSpec.Origin != (SourceInfo{}) {
			baseSpec.Origin = overrideSpec.Origin
		}
		base.Commands[name] = baseSpec
	}
}

func registerOrigins(cfg *Config, source SourceInfo) {
	for name, spec := range cfg.Commands {
		spec.Origin = source
		for actionName, action := range spec.Actions {
			action.Origin = source
			spec.Actions[actionName] = action
		}
		cfg.Commands[name] = spec
	}
}

func (l *Loader) describeSource(path string, module string) SourceInfo {
	cleaned := filepath.Clean(path)
	return SourceInfo{
		Module: module,
		File:   filepath.ToSlash(cleaned),
	}
}

func collectModuleYAML(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if strings.EqualFold(name, "scripts") {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext == ".yaml" || ext == ".yml" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func (l *Loader) loadDirectoryConfig(dir string) (*Config, error) {
	files, err := collectModuleYAML(dir)
	if err != nil {
		return nil, fmt.Errorf("遍历目录 %s 失败: %w", dir, err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("目录 %s 中未找到 YAML 配置", dir)
	}
	result := &Config{Commands: map[string]CommandSpec{}}
	moduleName := filepath.Base(dir)
	for _, file := range files {
		cfg, err := loadSingleConfig(file, l.describeSource(file, moduleName))
		if err != nil {
			return nil, fmt.Errorf("加载目录 %s 的配置 %s 失败: %w", moduleName, filepath.Base(file), err)
		}
		mergeConfig(result, cfg, mergeOptions{
			trackOverrides: true,
			label:          fmt.Sprintf("%s/%s", moduleName, filepath.Base(file)),
			collect: func(d Diagnostic) {
				l.diagnostics = append(l.diagnostics, d)
			},
		})
	}
	return result, nil
}
