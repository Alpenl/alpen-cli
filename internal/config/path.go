package config

import (
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
