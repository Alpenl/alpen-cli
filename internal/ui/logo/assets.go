package logo

import (
	_ "embed"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed logo.txt
var rawLogo string

//go:embed icons.yaml
var rawIcons []byte

var (
	logoOnce sync.Once
	logoData []string

	iconOnce sync.Once
	icons    map[string]string
)

// Lines 返回 ASCII Logo，每行作为一个元素
func Lines() []string {
	logoOnce.Do(func() {
		trimmed := strings.TrimRight(rawLogo, "\n")
		if trimmed == "" {
			logoData = nil
			return
		}
		logoData = strings.Split(trimmed, "\n")
	})
	return logoData
}

// Icon 返回指定名称的图标
func Icon(name string) string {
	loadIcons()
	if icons == nil {
		return ""
	}
	if value, ok := icons[name]; ok && value != "" {
		return value
	}
	if value, ok := icons["default"]; ok {
		return value
	}
	return ""
}

// Icons 返回所有图标的拷贝，避免被外部修改
func Icons() map[string]string {
	loadIcons()
	if len(icons) == 0 {
		return map[string]string{}
	}
	copy := make(map[string]string, len(icons))
	for k, v := range icons {
		copy[k] = v
	}
	return copy
}

func loadIcons() {
	iconOnce.Do(func() {
		if len(rawIcons) == 0 {
			icons = map[string]string{}
			return
		}
		var data map[string]string
		if err := yaml.Unmarshal(rawIcons, &data); err != nil {
			icons = map[string]string{}
			return
		}
		icons = data
	})
}
