package ui

import (
	"fmt"
	"io"

	"github.com/alpen/alpen-cli/internal/ui/logo"
)

// CommandInfo 命令信息
type CommandInfo struct {
	Name        string
	Description string
}

// RenderLogo 输出统一的 ASCII Logo
func RenderLogo(w io.Writer) {
	lines := logo.Lines()
	if len(lines) == 0 {
		return
	}
	for _, line := range lines {
		fmt.Fprintln(w, Cyan(line))
	}
	fmt.Fprintln(w, "")
}

// ShowWelcome 显示美化的欢迎页面
func ShowWelcome(w io.Writer, commands []CommandInfo) {
	RenderLogo(w)

	// 显示简介
	fmt.Fprintln(w, Gray("  团队脚本统一管理与执行工具"))
	fmt.Fprintln(w, "")

	// 显示核心命令
	MenuTitle(w, "核心命令")
	fmt.Fprintln(w, "")

	for _, c := range commands {
		fmt.Fprintf(w, "  %s  %s\n",
			Cyan(fmt.Sprintf("%-12s", c.Name)),
			Gray(c.Description))
	}

	fmt.Fprintln(w, "")
	Separator(w)
	fmt.Fprintln(w, "")

	// 显示快速开始提示
	fmt.Fprintf(w, "  %s 快速开始：%s\n",
		IconInfo(),
		Highlight("alpen ui"))
	fmt.Fprintf(w, "  %s 查看帮助：%s\n",
		IconInfo(),
		Highlight("alpen help"))

	fmt.Fprintln(w, "")
}
