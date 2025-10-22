package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alpen/alpen-cli/internal/config"
	"github.com/alpen/alpen-cli/internal/templates"
)

// HomeAssetsResult 描述 ~/.alpen/config 下的关键路径
type HomeAssetsResult struct {
	ConfigPath string
	ScriptsDir string
	TestsDir   string
}

// EnsureHomeAssets 创建 ~/.alpen/config 目录结构并生成示例资产
func EnsureHomeAssets(force bool) (*HomeAssetsResult, error) {
	if _, err := config.EnsureHomeStructure(); err != nil {
		return nil, err
	}
	configPath, err := config.DefaultConfigPath()
	if err != nil {
		return nil, err
	}
	scriptsDir, err := config.DefaultScriptsRoot()
	if err != nil {
		return nil, err
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

	commandTemplate, err := templates.DefaultCommands(map[string]string{
		"ScriptsRoot": filepath.ToSlash(scriptsDir),
	})
	if err != nil {
		return nil, err
	}
	if err := WriteFileIfNeeded(configPath, commandTemplate, 0o644, force); err != nil {
		return nil, err
	}

	configDir := filepath.Dir(configPath)
	if err := ensureDemoModule(configDir, force); err != nil {
		return nil, err
	}

	return &HomeAssetsResult{
		ConfigPath: configPath,
		ScriptsDir: scriptsDir,
		TestsDir:   testsDir,
	}, nil
}

// EnsureHomeReadme 在 ~/.alpen 下写入 README
func EnsureHomeReadme(configDir string, force bool) error {
	readmePath := filepath.Join(filepath.Dir(configDir), "README.md")
	if _, err := os.Stat(readmePath); err == nil && !force {
		return nil
	}
	return os.WriteFile(readmePath, []byte(templates.HomeReadme()), 0o644)
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

func WriteFileIfNeeded(path string, content string, perm os.FileMode, force bool) error {
	if _, err := os.Stat(path); err == nil && !force {
		return nil
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.WriteFile(path, []byte(content), perm)
}
