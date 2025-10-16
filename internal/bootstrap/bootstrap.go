package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alpen/alpen-cli/internal/config"
	"github.com/alpen/alpen-cli/internal/templates"
)

// GlobalAssetsResult 描述全局初始化后的关键路径
type GlobalAssetsResult struct {
	ConfigPath string
	ScriptsDir string
	TestsDir   string
}

// EnsureGlobalAssets 创建全局目录结构并生成示例资产
func EnsureGlobalAssets(cfg *config.GlobalConfig, force bool) (*GlobalAssetsResult, error) {
	if cfg == nil {
		return nil, errors.New("全局配置为空")
	}
	if _, err := config.EnsureGlobalStructure(); err != nil {
		return nil, err
	}
	scriptsDir := strings.TrimSpace(cfg.ScriptsRoot)
	if scriptsDir == "" {
		home, err := config.ResolveHomeDir()
		if err != nil {
			return nil, err
		}
		scriptsDir = filepath.Join(home, "config", "scripts")
		cfg.ScriptsRoot = scriptsDir
	}
	scriptsDir = filepath.Clean(scriptsDir)
	if strings.TrimSpace(cfg.DefaultConfigPath) == "" {
		cfg.DefaultConfigPath = filepath.Join(filepath.Dir(scriptsDir), "demo.yaml")
	}
	testsDir := filepath.Join(scriptsDir, "tests")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建脚本目录失败: %w", err)
	}
	if err := WriteFileIfNeeded(filepath.Join(scriptsDir, "demo.sh"), templates.DemoShell(), 0o755, force); err != nil {
		return nil, err
	}
	if err := WriteFileIfNeeded(filepath.Join(scriptsDir, "demo.py"), templates.DemoPython(), 0o755, force); err != nil {
		return nil, err
	}
	if err := WriteFileIfNeeded(filepath.Join(testsDir, "demo.sh"), templates.DemoTest(), 0o755, force); err != nil {
		return nil, err
	}

	commandTemplate, err := templates.GlobalCommands(map[string]string{
		"ScriptsRoot": filepath.ToSlash(scriptsDir),
	})
	if err != nil {
		return nil, err
	}
	if err := WriteFileIfNeeded(cfg.DefaultConfigPath, commandTemplate, 0o644, force); err != nil {
		return nil, err
	}

	configDir := filepath.Dir(cfg.DefaultConfigPath)
	if err := ensureDemoModule(configDir, force); err != nil {
		return nil, err
	}

	if len(cfg.SearchPaths) == 0 {
		cfg.SearchPaths = []string{configDir}
	}

	return &GlobalAssetsResult{
		ConfigPath: cfg.DefaultConfigPath,
		ScriptsDir: scriptsDir,
		TestsDir:   testsDir,
	}, nil
}

// EnsureLocalAssets 在项目目录生成示例资产
func EnsureLocalAssets(baseDir string, force bool) (string, error) {
	targetDir := baseDir
	if strings.TrimSpace(targetDir) == "" {
		var err error
		targetDir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	alpenDir := filepath.Join(targetDir, ".alpen")
	scriptsDir := filepath.Join(alpenDir, "scripts")
	testsDir := filepath.Join(scriptsDir, "tests")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		return "", fmt.Errorf("创建目录 %s 失败: %w", testsDir, err)
	}
	targetFile := filepath.Join(alpenDir, "demo.yaml")

	if err := WriteFileIfNeeded(filepath.Join(scriptsDir, "demo.sh"), templates.DemoShell(), 0o755, force); err != nil {
		return "", err
	}
	if err := WriteFileIfNeeded(filepath.Join(scriptsDir, "demo.py"), templates.DemoPython(), 0o755, force); err != nil {
		return "", err
	}
	if err := WriteFileIfNeeded(filepath.Join(testsDir, "demo.sh"), templates.DemoTest(), 0o755, force); err != nil {
		return "", err
	}
	if err := WriteFileIfNeeded(targetFile, templates.LocalCommands(), 0o644, force); err != nil {
		return "", err
	}

	if err := ensureDemoModule(alpenDir, force); err != nil {
		return "", err
	}
	return targetFile, nil
}

// EnsureGlobalReadme 在全局目录写入 README
func EnsureGlobalReadme(configDir string, force bool) error {
	readmePath := filepath.Join(filepath.Dir(configDir), "README.md")
	if _, err := os.Stat(readmePath); err == nil && !force {
		return nil
	}
	return os.WriteFile(readmePath, []byte(templates.GlobalReadme()), 0o644)
}

// PersistGlobalConfig 写入 global.yaml
func PersistGlobalConfig(cfg *config.GlobalConfig, force bool) error {
	if cfg == nil {
		return errors.New("全局配置为空")
	}
	home, err := config.ResolveHomeDir()
	if err != nil {
		return err
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

	content, err := templates.GlobalConfig(map[string]any{
		"DefaultConfigPath": filepath.ToSlash(cfg.DefaultConfigPath),
		"ScriptsRoot":       filepath.ToSlash(cfg.ScriptsRoot),
		"ConfigDir":         filepath.ToSlash(filepath.Dir(cfg.DefaultConfigPath)),
		"SearchPaths":       ToSlashSlice(cfg.SearchPaths),
	})
	if err != nil {
		return err
	}
	return WriteFileIfNeeded(filepath.Join(home, "global.yaml"), content, 0o644, force)
}

func ensureDemoModule(configDir string, force bool) error {
	moduleDir := filepath.Join(configDir, "demo.conf")
	scriptsDir := filepath.Join(moduleDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		return fmt.Errorf("创建演示模块目录失败: %w", err)
	}

	moduleScriptPath := filepath.Join(scriptsDir, "demo_module.sh")
	if err := WriteFileIfNeeded(moduleScriptPath, templates.DemoModuleScript(), 0o755, force); err != nil {
		return err
	}

	content, err := templates.DemoModuleConfig(map[string]string{
		"ModuleScript": filepath.ToSlash(moduleScriptPath),
	})
	if err != nil {
		return err
	}
	moduleConfigPath := filepath.Join(moduleDir, "100_demo.yaml")
	if err := WriteFileIfNeeded(moduleConfigPath, content, 0o644, force); err != nil {
		return err
	}
	return nil
}

// UniqueStrings 去重并保持顺序
func UniqueStrings(items []string) []string {
	seen := map[string]struct{}{}
	var result []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func WriteFileIfNeeded(path string, content string, perm os.FileMode, force bool) error {
	if _, err := os.Stat(path); err == nil && !force {
		return nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.WriteFile(path, []byte(content), perm)
}

func ToSlashSlice(paths []string) []string {
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		result = append(result, filepath.ToSlash(p))
	}
	return result
}
