package commands

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alpen/alpen-cli/internal/config"
	"github.com/alpen/alpen-cli/internal/executor"
	"github.com/alpen/alpen-cli/internal/ui"
)

const annotationDynamic = "alpen.dynamic"

// RegisterDynamicCommands 根据配置动态生成命令树
func RegisterDynamicCommands(root *cobra.Command, deps Dependencies, cfg *config.Config) error {
	if cfg == nil || cfg.Commands == nil {
		return nil
	}
	for _, name := range cfg.SortedCommandNames() {
		spec := cfg.Commands[name]
		cmd := buildTopLevelCommand(name, spec, deps)
		replaceCommand(root, cmd)
	}
	return nil
}

func buildTopLevelCommand(name string, spec config.CommandSpec, deps Dependencies) *cobra.Command {
	description := strings.TrimSpace(spec.Description)
	if description == "" {
		description = fmt.Sprintf("命令 %s", name)
	}

	cmd := &cobra.Command{
		Use:           name,
		Short:         description,
		Long:          buildCommandLongDescription(name, spec, description),
		SilenceUsage:  true,
		SilenceErrors: true,
		Annotations: map[string]string{
			annotationDynamic: "true",
		},
	}
	if alias := strings.TrimSpace(spec.Alias); alias != "" {
		cmd.Aliases = []string{alias}
	}
	if examples := buildCommandExamples(name, spec); examples != "" {
		cmd.Example = examples
	}

	cmd.AddCommand(buildCommandListSubcommand(name, spec))

	defaultCommand := strings.TrimSpace(spec.Command)
	if defaultCommand == "" {
		cmd.RunE = func(c *cobra.Command, _ []string) error {
			return c.Help()
		}
	} else {
		commandLine := defaultCommand
		cmd.RunE = func(c *cobra.Command, _ []string) error {
			return executeDynamic(c, deps, []string{name}, commandLine, c.Flags().Args())
		}
	}

	for _, actionName := range spec.SortedActionNames() {
		if actionName == "ls" {
			continue
		}
		action := spec.Actions[actionName]
		child := buildActionCommand(name, actionName, action, deps)
		cmd.AddCommand(child)
	}

	return cmd
}

func buildActionCommand(parent string, name string, spec config.ActionSpec, deps Dependencies) *cobra.Command {
	description := strings.TrimSpace(spec.Description)
	if description == "" {
		description = fmt.Sprintf("命令 %s %s", parent, name)
	}
	commandLine := strings.TrimSpace(spec.Command)

	cmd := &cobra.Command{
		Use:           name,
		Short:         description,
		Long:          buildActionLongDescription(parent, name, spec, description),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(c *cobra.Command, _ []string) error {
			return executeDynamic(c, deps, []string{parent, name}, commandLine, c.Flags().Args())
		},
	}

	if alias := strings.TrimSpace(spec.Alias); alias != "" {
		cmd.Aliases = []string{alias}
	}
	cmd.Example = buildActionExamples(parent, name)
	return cmd
}

func executeDynamic(cmd *cobra.Command, deps Dependencies, path []string, command string, args []string) error {
	if deps.Executor == nil {
		return fmt.Errorf("执行器未初始化")
	}
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("命令 %s 未配置可执行脚本", strings.Join(path, " "))
	}

	displayName := strings.Join(path, " ")
	writer := cmd.OutOrStdout()

	ui.BeginExecution(writer, displayName)

	req := executor.ScriptRequest{
		CommandPath: path,
		Command:     command,
		ExtraArgs:   args,
	}

	result, err := deps.Executor.Execute(cmd.Context(), req)

	ui.EndExecution(writer)
	if err != nil {
		ui.ExecutionSummary(writer, false, result.Duration, err)
		return wrapReportedError(err)
	}
	ui.ExecutionSummary(writer, true, result.Duration, nil)
	return nil
}

func replaceCommand(root *cobra.Command, cmd *cobra.Command) {
	for _, existing := range root.Commands() {
		if existing.Name() == cmd.Name() {
			if existing.Annotations != nil && existing.Annotations[annotationDynamic] == "true" {
				root.RemoveCommand(existing)
			} else {
				// 保留内置命令，避免用户配置覆盖
				return
			}
			break
		}
	}
	root.AddCommand(cmd)
}

func buildCommandLongDescription(name string, spec config.CommandSpec, fallback string) string {
	var sections []string
	if fallback != "" {
		sections = append(sections, fallback)
	}
	if alias := strings.TrimSpace(spec.Alias); alias != "" {
		sections = append(sections, fmt.Sprintf("别名：%s", alias))
	}
	if strings.TrimSpace(spec.Command) != "" {
		sections = append(sections, "默认执行：直接运行 `alpen "+name+"` 即可触发")
	}
	if len(spec.Actions) > 0 {
		sections = append(sections, "子命令：使用 `alpen "+name+" <action>` 调用具体动作")
	}
	sections = append(sections, "参数透传：使用 `--` 之后追加原生命令参数，例如 `alpen "+name+" -- --flag value`")
	return strings.Join(sections, "\n\n")
}

func buildCommandExamples(name string, spec config.CommandSpec) string {
	var examples []string
	if strings.TrimSpace(spec.Command) != "" {
		examples = append(examples, fmt.Sprintf("  alpen %s", name))
	}
	if actionNames := spec.SortedActionNames(); len(actionNames) > 0 {
		examples = append(examples, fmt.Sprintf("  alpen %s %s", name, actionNames[0]))
	}
	if len(examples) > 0 {
		examples = append(examples, fmt.Sprintf("  alpen %s -- --flag value", name))
	}
	return strings.Join(examples, "\n")
}

func buildActionLongDescription(parent, name string, spec config.ActionSpec, fallback string) string {
	var sections []string
	if fallback != "" {
		sections = append(sections, fallback)
	}
	if alias := strings.TrimSpace(spec.Alias); alias != "" {
		sections = append(sections, fmt.Sprintf("别名：%s", alias))
	}
	sections = append(sections, fmt.Sprintf("调用方式：`alpen %s %s`", parent, name))
	sections = append(sections, fmt.Sprintf("参数透传：`alpen %s %s -- --flag value`", parent, name))
	return strings.Join(sections, "\n\n")
}

func buildActionExamples(parent, name string) string {
	return fmt.Sprintf("  alpen %s %s\n  alpen %s %s -- --flag value", parent, name, parent, name)
}

func buildCommandListSubcommand(name string, spec config.CommandSpec) *cobra.Command {
	return &cobra.Command{
		Use:           "ls",
		Short:         fmt.Sprintf("查看 %s 下的命令列表", name),
		Long:          fmt.Sprintf("列出命令 %s 及其子命令的名称、别名与简介。", name),
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runCommandList(cmd.OutOrStdout(), name, spec)
		},
	}
}

func runCommandList(writer io.Writer, name string, spec config.CommandSpec) error {
	ui.MenuTitle(writer, name)
	fmt.Fprintln(writer, "")

	commandWidth := displayWidth(strings.TrimSpace(name))
	actionWidth := maxActionNameWidth(spec)
	writeCommandSummary(writer, name, spec, commandWidth, actionWidth)
	return nil
}
