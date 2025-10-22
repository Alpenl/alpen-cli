package logo

import (
	_ "embed"
	"strings"
	"sync"
)

//go:embed logo.txt
var rawLogo string

var (
	logoOnce sync.Once
	logoData []string
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
