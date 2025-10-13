package commands

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/alpen/alpen-cli/internal/config"
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
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, "组\t脚本\t描述")

	groupNames := make([]string, 0, len(cfg.Groups))
	for name := range cfg.Groups {
		groupNames = append(groupNames, name)
	}
	sort.Strings(groupNames)
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
		for _, scriptName := range scriptNames {
			script := group.Scripts[scriptName]
			fmt.Fprintf(w, "%s\t%s\t%s\n", groupName, scriptName, script.Description)
		}
	}
	return w.Flush()
}
