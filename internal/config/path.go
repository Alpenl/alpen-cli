package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ExpandPath 展开路径中的 ~ 与环境变量，并返回规范化结果
func ExpandPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}

	// 展开 ~ 前缀
	if strings.HasPrefix(p, "~") {
		if home, err := ResolveHomeDir(); err == nil {
			trimmed := strings.TrimPrefix(p, "~")
			trimmed = strings.TrimPrefix(trimmed, string(os.PathSeparator))
			p = filepath.Join(home, trimmed)
		}
	}

	// 展开环境变量
	p = os.ExpandEnv(p)

	return filepath.Clean(p)
}

// NormalizeConfigPath 将配置文件路径归一化至 ~/.alpen 下
func NormalizeConfigPath(raw string) (string, error) {
	home, err := ResolveHomeDir()
	if err != nil {
		return "", err
	}
	defaultPath := filepath.Join(home, "config", "demo.yaml")
	return normalizeWithinHome(raw, defaultPath, home)
}

// NormalizeScriptsRoot 将脚本目录路径归一化至 ~/.alpen 下
func NormalizeScriptsRoot(raw string) (string, error) {
	home, err := ResolveHomeDir()
	if err != nil {
		return "", err
	}
	defaultPath := filepath.Join(home, "config", "scripts")
	return normalizeWithinHome(raw, defaultPath, home)
}

func normalizeWithinHome(raw string, fallback string, home string) (string, error) {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		candidate = fallback
	}
	if strings.TrimSpace(candidate) == "" {
		return "", fmt.Errorf("路径不能为空")
	}

	expanded := ExpandPath(candidate)
	if !filepath.IsAbs(expanded) {
		expanded = filepath.Join(home, expanded)
	}
	absCandidate, err := filepath.Abs(expanded)
	if err != nil {
		return "", fmt.Errorf("解析路径失败: %w", err)
	}
	absHome, err := filepath.Abs(home)
	if err != nil {
		return "", fmt.Errorf("解析用户目录失败: %w", err)
	}
	rel, err := filepath.Rel(absHome, absCandidate)
	if err != nil {
		return "", fmt.Errorf("计算路径关系失败: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("路径 %s 不在用户目录 %s 内", absCandidate, absHome)
	}
	return absCandidate, nil
}
