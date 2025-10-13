package commands

import (
	"fmt"
	"sort"

	"github.com/alpen/alpen-cli/internal/config"
	"github.com/alpen/alpen-cli/internal/ui"
	"github.com/spf13/cobra"
)

// NewListCommand 创建 list 子命令
func NewListCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出配置中所有可用脚本",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(cmd, deps.Loader)
			if err != nil {
				return err
			}
			groupFilter, _ := cmd.Flags().GetString("group")
			return renderList(cmd, cfg, groupFilter)
		},
	}
	cmd.Flags().StringP("group", "g", "", "仅显示指定分组中的脚本")
	return cmd
}

func renderList(cmd *cobra.Command, cfg *config.Config, filter string) error {
	writer := cmd.OutOrStdout()

	groupNames := make([]string, 0, len(cfg.Groups))
	for name := range cfg.Groups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)

	if len(groupNames) == 0 {
		ui.Warning(writer, "当前配置未定义任何脚本组")
		return nil
	}

	fmt.Fprintln(writer, "")
	ui.Title(writer, "可用脚本列表")
	ui.Separator(writer)

	foundAny := false
	for _, groupName := range groupNames {
		if filter != "" && filter != groupName {
			continue
		}
		group := cfg.Groups[groupName]
		scriptNames := make([]string, 0, len(group.Scripts))
		for name := range group.Scripts {
			scriptNames = append(scriptNames, name)
		}
		sort.Strings(scriptNames)

		if len(scriptNames) == 0 {
			continue
		}

		foundAny = true
		fmt.Fprintln(writer, "")
		fmt.Fprintf(writer, "%s %s\n", ui.Cyan("▶"), ui.Highlight(groupName))
		if group.Description != "" {
			fmt.Fprintf(writer, "  %s\n", ui.Gray(group.Description))
		}
		fmt.Fprintln(writer, "")

		for _, scriptName := range scriptNames {
			script := group.Scripts[scriptName]
			desc := script.Description
			if desc == "" {
				desc = ui.Gray("(无描述)")
			} else {
				desc = ui.Gray(desc)
			}
			fmt.Fprintf(writer, "  • %s  %s\n", ui.Cyan(groupName+"/"+scriptName), desc)
		}
	}

	if !foundAny {
		if filter != "" {
			ui.Warning(writer, "分组 '%s' 不存在或没有脚本", filter)
		} else {
			ui.Warning(writer, "当前配置未定义任何脚本")
		}
	}

	fmt.Fprintln(writer, "")
	return nil
}
