package commands

import (
	"fmt"
	"strings"

	"github.com/alpen/alpen-cli/internal/config"
	"github.com/alpen/alpen-cli/internal/executor"
	"github.com/alpen/alpen-cli/internal/ui"
	"github.com/spf13/cobra"
)

// NewRunCommand 创建 run 子命令
func NewRunCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <脚本名 | 组/脚本名> [-- <额外参数>]",
		Short: "执行配置中的指定脚本",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(cmd, deps.Loader)
			if err != nil {
				return err
			}
			return runScript(cmd, cfg, deps, cmd.Flags().Args())
		},
	}
	cmd.Flags().Bool("dry-run", false, "仅打印将要执行的命令，不实际执行")
	cmd.Flags().StringToString("env", map[string]string{}, "为脚本注入额外环境变量，格式为 KEY=VALUE")
	cmd.Flags().String("group", "", "当脚本名在多个分组重复时使用该选项指定分组")
	cmd.Flags().String("dir", "", "指定脚本执行时的工作目录，默认为当前目录")
	return cmd
}

func runScript(cmd *cobra.Command, cfg *config.Config, deps Dependencies, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("请提供脚本名称")
	}
	groupOverride, _ := cmd.Flags().GetString("group")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	envOverrides, _ := cmd.Flags().GetStringToString("env")
	workDir, _ := cmd.Flags().GetString("dir")

	scriptToken := args[0]

	groupName, scriptName, template, err := resolveScript(cfg, scriptToken, groupOverride)
	if err != nil {
		return err
	}

	scriptArgs := extractScriptArgs(cmd, args)

	req := executor.ScriptRequest{
		GroupName:  groupName,
		ScriptName: scriptName,
		Template:   template,
		ExtraArgs:  scriptArgs,
		ExtraEnv:   envOverrides,
		WorkingDir: workDir,
		DryRun:     dryRun,
	}

	writer := cmd.OutOrStdout()

	if dryRun {
		fmt.Fprintln(writer, "")
		ui.Info(writer, "Dry-run 模式: %s/%s", groupName, scriptName)
		ui.Separator(writer)
	} else {
		fmt.Fprintln(writer, "")
		ui.Executing(writer, fmt.Sprintf("%s/%s", groupName, scriptName))
		ui.Separator(writer)
	}

	result, err := deps.Executor.Execute(cmd.Context(), req)

	if !dryRun {
		fmt.Fprintln(writer, "")
		ui.Separator(writer)
	}

	if err != nil {
		ui.Error(writer, "脚本执行失败")
		ui.Duration(writer, result.Duration.String())
		return err
	}

	if !dryRun {
		ui.Success(writer, "脚本执行完成")
		ui.Duration(writer, result.Duration.String())
	} else {
		ui.Success(writer, "Dry-run 模式完成")
	}
	return nil
}

func resolveScript(cfg *config.Config, scriptToken string, groupOverride string) (string, string, config.ScriptTemplate, error) {
	if strings.Contains(scriptToken, "/") {
		groupName, scriptName, tmpl, err := cfg.FindScriptQualified(scriptToken)
		return groupName, scriptName, tmpl, err
	}
	if groupOverride != "" {
		group, ok := cfg.Groups[groupOverride]
		if !ok {
			return "", "", config.ScriptTemplate{}, fmt.Errorf("指定的分组 %s 不存在", groupOverride)
		}
		tmpl, ok := group.Scripts[scriptToken]
		if !ok {
			return "", "", config.ScriptTemplate{}, fmt.Errorf("分组 %s 中不存在脚本 %s", groupOverride, scriptToken)
		}
		return groupOverride, scriptToken, tmpl, nil
	}
	groupName, tmpl, err := cfg.FindScript(scriptToken)
	if err != nil {
		return "", "", config.ScriptTemplate{}, err
	}
	return groupName, scriptToken, tmpl, nil
}

func extractScriptArgs(cmd *cobra.Command, args []string) []string {
	if len(args) <= 1 {
		return nil
	}
	dashIdx := cmd.Flags().ArgsLenAtDash()
	if dashIdx >= 0 {
		start := dashIdx
		if start == 0 {
			start = 1
		}
		if start < len(args) {
			return args[start:]
		}
		return nil
	}
	return args[1:]
}
