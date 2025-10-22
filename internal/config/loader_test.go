package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoaderLoadWithEnvOverride(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "demo.yaml")
	envPath := filepath.Join(dir, "demo.dev.yaml")

	baseContent := []byte(`
commands:
  system:
    description: 系统管理
    command: echo base
`)
	envContent := []byte(`
commands:
  system:
    command: echo override
`)

	if err := os.WriteFile(basePath, baseContent, 0o644); err != nil {
		t.Fatalf("write base config failed: %v", err)
	}
	if err := os.WriteFile(envPath, envContent, 0o644); err != nil {
		t.Fatalf("write env config failed: %v", err)
	}

	loader := NewLoader(dir)
	cfg, err := loader.Load("demo.yaml", "dev")
	if err != nil {
		t.Fatalf("load config failed: %v", err)
	}

	if cfg.Commands["system"].Command != "echo override" {
		t.Fatalf("expected override command, got %s", cfg.Commands["system"].Command)
	}
}

func TestLoaderLoadWithModules(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "demo.yaml")
	moduleDir := filepath.Join(dir, "deploy.conf")

	baseContent := []byte(`
commands:
  system:
    description: 基础命令
    command: echo base
`)
	moduleContent := []byte(`
commands:
  deploy:
    description: 部署命令
    command: echo deploy
`)

	if err := os.WriteFile(basePath, baseContent, 0o644); err != nil {
		t.Fatalf("write base config failed: %v", err)
	}
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("create module dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "001_deploy.yaml"), moduleContent, 0o644); err != nil {
		t.Fatalf("write module config failed: %v", err)
	}

	loader := NewLoader(dir)
	baseCfg, err := loader.Load("demo.yaml", "")
	if err != nil {
		t.Fatalf("load base config failed: %v", err)
	}
	if _, ok := baseCfg.Commands["deploy"]; ok {
		t.Fatalf("unexpected deploy command in base config")
	}

	moduleCfg, err := loader.Load("deploy.conf", "")
	if err != nil {
		t.Fatalf("load module config failed: %v", err)
	}
	if _, ok := moduleCfg.Commands["deploy"]; !ok {
		t.Fatalf("expected deploy command in module config")
	}
	if moduleCfg.Commands["deploy"].Command != "echo deploy" {
		t.Fatalf("expected deploy command from module, got %s", moduleCfg.Commands["deploy"].Command)
	}
	if len(loader.Diagnostics()) != 0 {
		t.Fatalf("expected no diagnostics for new command, got %d", len(loader.Diagnostics()))
	}
	if len(moduleCfg.Diagnostics) != 0 {
		t.Fatalf("expected module diagnostics empty, got %d", len(moduleCfg.Diagnostics))
	}
}

func TestLoaderModuleOverrideProducesError(t *testing.T) {
	dir := t.TempDir()
	basePath := filepath.Join(dir, "demo.yaml")
	moduleDir := filepath.Join(dir, "deploy.conf")

	baseContent := []byte(`
commands:
  deploy:
    description: 原始部署
    command: echo base
`)
	moduleBase := []byte(`
commands:
  deploy:
    command: echo base
`)
	moduleOverride := []byte(`
commands:
  deploy:
    command: echo override
`)

	if err := os.WriteFile(basePath, baseContent, 0o644); err != nil {
		t.Fatalf("write base config failed: %v", err)
	}
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("create module dir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "100_base.yaml"), moduleBase, 0o644); err != nil {
		t.Fatalf("write module base failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "200_override.yaml"), moduleOverride, 0o644); err != nil {
		t.Fatalf("write module override failed: %v", err)
	}

	loader := NewLoader(dir)
	_, err := loader.Load("deploy.conf", "")
	// 应该因为配置冲突而返回错误
	if err == nil {
		t.Fatalf("expected error for config conflict, got nil")
	}
	if !strings.Contains(err.Error(), "冲突") {
		t.Fatalf("expected error message to mention conflict, got: %v", err)
	}
}
