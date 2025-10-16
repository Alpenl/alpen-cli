package commands

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/spf13/cobra"

	"github.com/alpen/alpen-cli/internal/config"
	"github.com/alpen/alpen-cli/internal/ui"
)

// NewListCommand 创建 ls 子命令，用于浏览配置文件中定义的命令
func NewListCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ls",
		Short:         "查看配置中定义的命令列表",
		Long:          "列出当前配置文件中的所有顶层命令，展示名称、别名与简介，便于快速浏览自定义命令。",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRootList(cmd, deps)
		},
	}
	return cmd
}

func runRootList(cmd *cobra.Command, deps Dependencies) error {
	if deps.Loader == nil {
		return fmt.Errorf("配置加载器未初始化")
	}
	configPath, envName, err := resolveConfigFlags(cmd)
	if err != nil {
		return err
	}
	cfg, err := deps.Loader.Load(configPath, envName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			writer := cmd.OutOrStdout()
			ui.Warning(writer, "未检测到命令配置文件 %s", ui.Highlight(configPath))
			ui.Info(writer, "执行 %s 可生成示例结构", ui.Highlight("alpen init"))
			return nil
		}
		return err
	}

	writer := cmd.OutOrStdout()

	if cfg == nil || len(cfg.Commands) == 0 {
		ui.Warning(writer, "当前配置未包含自定义命令")
		return nil
	}

	renderDiagnostics(writer, cfg.Diagnostics)

	renderConfigSummary(writer, cfg, configPath)
	return nil
}

func renderConfigSummary(writer io.Writer, cfg *config.Config, configPath string) {
	fileName := filepath.Base(configPath)
	if fileName == "" {
		fileName = configPath
	}

	ui.Title(writer, fileName)
	fmt.Fprintf(writer, "%s %s\n", ui.Gray("配置路径:"), ui.Cyan(configPath))
	fmt.Fprintln(writer, "")
	fmt.Fprintln(writer, ui.Gray("命令列表:"))
	fmt.Fprintln(writer, "")

	printCommandSummaries(writer, cfg)
}

func printCommandSummaries(writer io.Writer, cfg *config.Config) {
	names := cfg.SortedCommandNames()
	if len(names) == 0 {
		return
	}
	maxCommandWidth := maxNameWidth(names)
	maxActionWidth := 0
	for _, name := range names {
		spec := cfg.Commands[name]
		if w := maxActionNameWidth(spec); w > maxActionWidth {
			maxActionWidth = w
		}
	}
	for idx, name := range names {
		spec := cfg.Commands[name]
		writeCommandSummary(writer, name, spec, maxCommandWidth, maxActionWidth)
		if idx < len(names)-1 {
			fmt.Fprintln(writer, "")
		}
	}
}

// writeCommandSummary 输出单个命令及其子命令的简介，缩进展示层级结构
func writeCommandSummary(writer io.Writer, name string, spec config.CommandSpec, commandWidth int, actionWidth int) {
	fmt.Fprintln(writer, formatEntry("  ", name, spec.Alias, spec.Description, commandWidth))
	if len(spec.Actions) == 0 {
		return
	}
	for _, actionName := range spec.SortedActionNames() {
		if actionName == "ls" {
			continue
		}
		action := spec.Actions[actionName]
		fmt.Fprintln(writer, formatEntry("    ", actionName, action.Alias, action.Description, actionWidth))
	}
}

func formatEntry(prefix, name, alias, description string, width int) string {
	nameCell := padRight(strings.TrimSpace(name), width)
	var meta []string
	if aliasText := strings.TrimSpace(alias); aliasText != "" {
		meta = append(meta, fmt.Sprintf("别名：%s", aliasText))
	}
	if desc := strings.TrimSpace(description); desc != "" {
		meta = append(meta, desc)
	}
	if len(meta) == 0 {
		return fmt.Sprintf("%s%s", prefix, nameCell)
	}
	return fmt.Sprintf("%s%s  %s", prefix, nameCell, strings.Join(meta, " · "))
}

func padRight(value string, width int) string {
	if width <= 0 {
		return value
	}
	length := displayWidth(value)
	if length >= width {
		return value
	}
	return value + strings.Repeat(" ", width-length)
}

func displayWidth(value string) int {
	return utf8.RuneCountInString(value)
}

func maxNameWidth(names []string) int {
	max := 0
	for _, name := range names {
		if length := displayWidth(strings.TrimSpace(name)); length > max {
			max = length
		}
	}
	return max
}

func maxActionNameWidth(spec config.CommandSpec) int {
	max := 0
	for _, actionName := range spec.SortedActionNames() {
		if actionName == "ls" {
			continue
		}
		if length := displayWidth(strings.TrimSpace(actionName)); length > max {
			max = length
		}
	}
	return max
}
