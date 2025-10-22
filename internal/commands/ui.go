package commands

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"

	"github.com/alpen/alpen-cli/internal/config"
	"github.com/alpen/alpen-cli/internal/executor"
	"github.com/alpen/alpen-cli/internal/ui"
)

// NewUICommand 创建 UI 命令入口
func NewUICommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "ui",
		Aliases: []string{"menu", "interactive"},
		Short:   "交互式命令导航",
		Long:    "以交互式列表的方式浏览命令树，选择后直接执行脚本，适合新成员快速上手。",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUI(cmd, deps)
		},
	}
	return cmd
}

type menuOption struct {
	Label       string
	Description string
	Path        []string
	Command     string
}

func runUI(cmd *cobra.Command, deps Dependencies) error {
	session, err := newUISession(cmd, deps)
	if err != nil || session == nil {
		return err
	}
	return session.run()
}

func resolveConfigFlags(cmd *cobra.Command) (string, string, error) {
	root := cmd.Root()
	configPath, err := root.PersistentFlags().GetString("config")
	if err != nil {
		return "", "", err
	}
	envName, err := root.PersistentFlags().GetString("environment")
	if err != nil {
		return "", "", err
	}
	return configPath, envName, nil
}

func buildMenuOptions(cfg *config.Config) []menuOption {
	if cfg == nil || cfg.Commands == nil {
		return nil
	}
	var options []menuOption
	for _, name := range cfg.SortedCommandNames() {
		spec := cfg.Commands[name]
		path := []string{name}
		if command := strings.TrimSpace(spec.Command); command != "" {
			label := name
			description := strings.TrimSpace(spec.Description)
			if alias := strings.TrimSpace(spec.Alias); alias != "" {
				label = fmt.Sprintf("%s (%s)", name, alias)
			}
			options = append(options, menuOption{
				Label:       label,
				Description: strings.TrimSpace(description),
				Path:        path,
				Command:     command,
			})
		}
		for _, actionName := range spec.SortedActionNames() {
			action := spec.Actions[actionName]
			label := fmt.Sprintf("%s %s", name, actionName)
			description := strings.TrimSpace(action.Description)
			if alias := strings.TrimSpace(action.Alias); alias != "" {
				label = fmt.Sprintf("%s %s (%s)", name, actionName, alias)
			}
			options = append(options, menuOption{
				Label:       label,
				Description: strings.TrimSpace(description),
				Path:        []string{name, actionName},
				Command:     action.Command,
			})
		}
	}
	return options
}

func appendDescriptor(description, extra string) string {
	description = strings.TrimSpace(description)
	extra = strings.TrimSpace(extra)
	switch {
	case description == "" && extra == "":
		return ""
	case description == "":
		return extra
	case extra == "":
		return description
	default:
		return fmt.Sprintf("%s（%s）", description, extra)
	}
}

// buildSurveyChoices 将 menuOption 转换为 survey 显示的选项
func buildSurveyChoices(options []menuOption) []string {
	choices := make([]string, 0, len(options)+1)

	var lastTopLevel string
	for _, option := range options {
		topLevel := option.Path[0]

		// 如果是新的主命令分组，添加分组标题（仅用于子命令）
		if len(option.Path) > 1 && topLevel != lastTopLevel {
			lastTopLevel = topLevel
		}

		// 根据命令类型添加不同前缀
		var prefix string
		if len(option.Path) == 1 {
			// 主命令默认动作
			prefix = "▪"
		} else {
			// 子命令
			prefix = "  ·"
		}

		choiceText := fmt.Sprintf("%s %s", prefix, option.Label)
		choices = append(choices, choiceText)
	}

	// 添加退出选项
	choices = append(choices, "✘ 退出")
	return choices
}

func promptExtraArgs(reader *bufio.Reader, writer io.Writer) ([]string, error) {
	ui.Prompt(writer, "额外参数（可选，可使用引号保留空格）: ")
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}
	parts, err := shellquote.Split(line)
	if err != nil {
		return nil, fmt.Errorf("参数格式错误: %w", err)
	}
	return parts, nil
}

type uiSession struct {
	cmd     *cobra.Command
	deps    Dependencies
	reader  *bufio.Reader
	writer  io.Writer
	options []menuOption
}

func newUISession(cmd *cobra.Command, deps Dependencies) (*uiSession, error) {
	ensureMenuSelectTemplate()
	if deps.Loader == nil {
		return nil, fmt.Errorf("配置加载器未初始化")
	}
	if deps.Executor == nil {
		return nil, fmt.Errorf("执行器未初始化")
	}

	writer := cmd.OutOrStdout()

	configPath, envName, err := resolveConfigFlags(cmd)
	if err != nil {
		return nil, err
	}

	cfg, err := deps.Loader.Load(configPath, envName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			ui.Warning(writer, "未检测到命令配置文件 %s", ui.Highlight(configPath))
			ui.Info(writer, "执行 %s 可生成默认配置", ui.Highlight("alpen init"))
			return nil, nil
		}
		return nil, err
	}

	renderDiagnostics(writer, cfg.Diagnostics)

	options := buildMenuOptions(cfg)
	if len(options) == 0 {
		ui.Warning(writer, "当前配置未包含可执行命令")
		return nil, nil
	}

	return &uiSession{
		cmd:     cmd,
		deps:    deps,
		reader:  bufio.NewReader(cmd.InOrStdin()),
		writer:  writer,
		options: options,
	}, nil
}

func (s *uiSession) run() error {
	s.renderIntro()

	for {
		index, err := s.promptMenuSelection()
		if err != nil {
			if errors.Is(err, io.EOF) || err.Error() == "interrupt" {
				s.renderExitMessage()
				return nil
			}
			ui.Warning(s.writer, "%v", err)
			continue
		}

		// 退出选项的索引是 len(options)
		if index >= len(s.options) {
			s.renderExitMessage()
			return nil
		}

		cont, err := s.executeOption(s.options[index])
		if err != nil {
			return err
		}
		if !cont {
			return nil
		}
	}
}

func (s *uiSession) renderIntro() {
	fmt.Fprintln(s.writer, "")
	ui.Banner(s.writer, "☰ Alpen 命令导航")
	fmt.Fprintln(s.writer, "")
}

func (s *uiSession) renderExitMessage() {
	fmt.Fprintln(s.writer, "")
	fmt.Fprintln(s.writer, "")
	fmt.Fprintln(s.writer, ui.Yellow("    已退出"))
}

func (s *uiSession) promptMenuSelection() (int, error) {
	choices := buildSurveyChoices(s.options)
	prompt := &survey.Select{
		Message: "选择命令 (↑/↓ 导航 | / 搜索 | Enter 确认)",
		Options: choices,
		Description: func(value string, index int) string {
			// 退出选项不显示描述
			if index >= len(s.options) {
				return ""
			}
			return s.options[index].Description
		},
		PageSize: 15,
		Filter:   s.buildPrefixFilter(),
		Help:     "",
	}
	var selectedIndex int
	if err := survey.AskOne(prompt, &selectedIndex); err != nil {
		return -1, err
	}
	return selectedIndex, nil
}

func (s *uiSession) buildPrefixFilter() func(filter string, value string, index int) bool {
	return func(filter string, value string, index int) bool {
		trimmed := strings.TrimSpace(filter)
		if trimmed == "" {
			return true
		}

		lowerValue := strings.ToLower(value)
		if strings.HasPrefix(trimmed, "/") {
			prefix := strings.TrimLeft(trimmed, "/")
			prefix = strings.ToLower(strings.TrimSpace(prefix))
			if prefix == "" {
				return true
			}
			// 退出选项
			if index >= len(s.options) {
				return strings.HasPrefix(strings.ToLower("退出"), prefix)
			}
			option := s.options[index]
			joinedPath := strings.ToLower(strings.Join(option.Path, " "))
			topLevel := ""
			if len(option.Path) > 0 {
				topLevel = strings.ToLower(option.Path[0])
			}
			return strings.HasPrefix(joinedPath, prefix) || strings.HasPrefix(topLevel, prefix)
		}

		return strings.Contains(lowerValue, strings.ToLower(trimmed))
	}
}

func (s *uiSession) executeOption(option menuOption) (bool, error) {
	if strings.TrimSpace(option.Command) == "" {
		ui.Warning(s.writer, "命令 %s 未配置可执行脚本", ui.Highlight(strings.Join(option.Path, " ")))
		return true, nil
	}

	extraArgs, err := promptExtraArgs(s.reader, s.writer)
	if err != nil {
		if errors.Is(err, io.EOF) {
			fmt.Fprintln(s.writer)
			return false, nil
		}
		ui.Error(s.writer, "解析参数失败: %v", err)
		return true, nil
	}

	displayName := strings.Join(option.Path, " ")
	ui.BeginExecution(s.writer, displayName)

	req := executor.ScriptRequest{
		CommandPath: option.Path,
		Command:     option.Command,
		ExtraArgs:   extraArgs,
	}

	result, execErr := s.deps.Executor.Execute(s.cmd.Context(), req)

	ui.EndExecution(s.writer)
	ui.ExecutionSummary(s.writer, execErr == nil, result.Duration, execErr)
	fmt.Fprintln(s.writer, "")

	return true, nil
}
