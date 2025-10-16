package ui

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/alpen/alpen-cli/internal/ui/logo"
)

// ANSI 转义码颜色定义
const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	gray   = "\033[90m"
	bold   = "\033[1m"
)

// 检测是否支持彩色输出
var colorEnabled = checkColorSupport()

func checkColorSupport() bool {
	// Windows 下需要检测终端类型
	if runtime.GOOS == "windows" {
		// 简化：假设 Windows 终端支持 ANSI（Windows 10+ 默认支持）
		return os.Getenv("TERM") != "dumb"
	}
	// Unix-like 系统检测终端
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	return true
}

// colorize 为文本添加颜色
func colorize(color, text string) string {
	if !colorEnabled {
		return text
	}
	return color + text + reset
}

// Success 输出成功消息（绿色）
func Success(w io.Writer, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(w, colorize(green, iconPrefix("success")+msg))
}

// Error 输出错误消息（红色）
func Error(w io.Writer, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(w, colorize(red, iconPrefix("error")+msg))
}

// Warning 输出警告消息（黄色）
func Warning(w io.Writer, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(w, colorize(yellow, iconPrefix("warning")+msg))
}

// Info 输出信息消息（蓝色）
func Info(w io.Writer, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(w, colorize(cyan, iconPrefix("info")+msg))
}

// Prompt 输出提示符（青色加粗）
func Prompt(w io.Writer, text string) {
	fmt.Fprint(w, colorize(cyan+bold, text))
}

// Title 输出标题（粗体）
func Title(w io.Writer, text string) {
	fmt.Fprintln(w, colorize(bold, text))
}

// Separator 输出分隔线
func Separator(w io.Writer) {
	fmt.Fprintln(w, colorize(gray, "────────────────────────────────────────"))
}

// MenuTitle 输出菜单标题
func MenuTitle(w io.Writer, title string) {
	Separator(w)
	fmt.Fprintln(w, colorize(cyan+bold, "  "+title))
	Separator(w)
}

// MenuItem 输出菜单项
func MenuItem(w io.Writer, index int, label, description string) {
	indexStr := colorize(cyan+bold, fmt.Sprintf("%d.", index))
	if description != "" && description != label {
		fmt.Fprintf(w, "  %s %s %s\n", indexStr, label, colorize(gray, "- "+description))
	} else {
		fmt.Fprintf(w, "  %s %s\n", indexStr, label)
	}
}

// Executing 输出执行提示（带动画效果）
func Executing(w io.Writer, scriptName string) {
	fmt.Fprintln(w, colorize(yellow, iconPrefix("executing")+"正在执行: "+scriptName))
}

// Duration 输出耗时（灰色）
func Duration(w io.Writer, duration string) {
	fmt.Fprintln(w, colorize(gray, "  耗时: "+duration))
}

// BeginExecution 输出统一的执行开始框架
func BeginExecution(w io.Writer, scriptName string) {
	name := strings.TrimSpace(scriptName)
	if name == "" {
		name = "未命名命令"
	}
	fmt.Fprintln(w, "")
	Executing(w, name)
	Separator(w)
}

// EndExecution 输出统一的执行结束分隔
func EndExecution(w io.Writer) {
	fmt.Fprintln(w, "")
	Separator(w)
}

// ExecutionSummary 输出统一的脚本执行摘要
func ExecutionSummary(w io.Writer, success bool, duration time.Duration, execErr error) {
	if success {
		fmt.Fprintln(w, colorize(green, iconPrefix("success")+"命令执行完成"))
	} else {
		fmt.Fprintln(w, colorize(red, iconPrefix("error")+"命令执行失败"))
	}

	Duration(w, duration.String())

	if success || execErr == nil {
		return
	}

	message := strings.TrimSpace(execErr.Error())
	if message == "" {
		return
	}

	lines := strings.Split(message, "\n")
	for index, line := range lines {
		prefix := "  错误: "
		if index > 0 {
			prefix = "        "
		}
		fmt.Fprintln(w, colorize(red, prefix+strings.TrimSpace(line)))
	}
}

// Highlight 高亮文本（粗体）
func Highlight(text string) string {
	return colorize(bold, text)
}

// Gray 灰色文本
func Gray(text string) string {
	return colorize(gray, text)
}

// Red 红色文本
func Red(text string) string {
	return colorize(red, text)
}

// Green 绿色文本
func Green(text string) string {
	return colorize(green, text)
}

// Yellow 黄色文本
func Yellow(text string) string {
	return colorize(yellow, text)
}

// Cyan 青色文本
func Cyan(text string) string {
	return colorize(cyan, text)
}

// IconSuccess 返回成功图标
func IconSuccess() string { return logo.Icon("success") }

// IconError 返回错误图标
func IconError() string { return logo.Icon("error") }

// IconWarning 返回警告图标
func IconWarning() string { return logo.Icon("warning") }

// IconInfo 返回信息图标
func IconInfo() string { return logo.Icon("info") }

// IconExecuting 返回执行图标
func IconExecuting() string { return logo.Icon("executing") }

// IconMenu 返回菜单图标
func IconMenu() string { return logo.Icon("menu") }

// IconCommand 返回命令图标
func IconCommand() string { return logo.Icon("command") }

// IconAction 返回动作图标
func IconAction() string { return logo.Icon("action") }

// IconSearch 返回搜索图标
func IconSearch() string { return logo.Icon("search") }

// IconExit 返回退出图标
func IconExit() string { return logo.Icon("exit") }

func iconPrefix(name string) string {
	icon := strings.TrimSpace(logo.Icon(name))
	if icon == "" {
		return ""
	}
	return icon + " "
}

// Banner 输出简洁标题横幅
func Banner(w io.Writer, title string) {
	if strings.TrimSpace(title) == "" {
		return
	}
	fmt.Fprintln(w, "")
	lineWidth := len([]rune(title)) + 4
	if lineWidth < 12 {
		lineWidth = 12
	}
	fmt.Fprintln(w, colorize(cyan+bold, "  "+title))
	fmt.Fprintln(w, colorize(gray, "  "+strings.Repeat("─", lineWidth)))
	fmt.Fprintln(w, "")
}

// Box 输出带边框的文本块
func Box(w io.Writer, content string) {
	lines := strings.Split(content, "\n")
	maxLen := 0
	for _, line := range lines {
		if len(line) > maxLen {
			maxLen = len(line)
		}
	}

	border := strings.Repeat("─", maxLen+2)
	fmt.Fprintln(w, colorize(gray, "┌"+border+"┐"))
	for _, line := range lines {
		padding := maxLen - len(line)
		fmt.Fprintf(w, colorize(gray, "│")+" %s%s "+colorize(gray, "│")+"\n",
			line, strings.Repeat(" ", padding))
	}
	fmt.Fprintln(w, colorize(gray, "└"+border+"┘"))
}

// CommandItem 输出格式化的命令项（用于菜单显示）
func CommandItem(label, description string) string {
	if description != "" {
		return fmt.Sprintf("%s %s %s",
			colorize(cyan+bold, label),
			colorize(gray, "│"),
			colorize(gray, description))
	}
	return colorize(cyan+bold, label)
}
