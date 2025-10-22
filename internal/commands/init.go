package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alpen/alpen-cli/internal/bootstrap"
	"github.com/alpen/alpen-cli/internal/config"
	"github.com/alpen/alpen-cli/internal/ui"
)

// NewInitCommand 创建 init 子命令
func NewInitCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "初始化默认命令配置文件",
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")
			return initHomeConfig(force, cmd)
		},
	}
	cmd.Flags().Bool("force", false, "若文件已存在则覆盖")
	return cmd
}

func initHomeConfig(force bool, cmd *cobra.Command) error {
	writer := cmd.OutOrStdout()

	configPath, err := config.DefaultConfigPath()
	if err != nil {
		return err
	}

	alreadyExists := !force && fileExists(configPath)

	result, err := bootstrap.EnsureHomeAssets(force)
	if err != nil {
		return err
	}

	if err := bootstrap.EnsureHomeReadme(filepath.Dir(result.ConfigPath), force); err != nil {
		ui.Warning(writer, "写入 README 失败: %v", err)
	}

	if err := config.SaveActiveConfigPath(result.ConfigPath); err != nil {
		ui.Warning(writer, "无法记录激活配置: %v", err)
	}

	// 输出美化的结果
	fmt.Fprintln(writer, "")
	if alreadyExists {
		ui.Info(writer, "检测到示例配置已存在，未执行覆盖操作")
		fmt.Fprintln(writer, "")
		ui.KeyValue(writer, "配置文件", result.ConfigPath)
		ui.KeyValue(writer, "脚本目录", result.ScriptsDir)
		fmt.Fprintln(writer, "")
		ui.Info(writer, "如需覆盖示例配置，可使用 %s", ui.Highlight("alpen init --force"))
	} else {
		ui.Success(writer, "已在用户目录生成示例配置")
		fmt.Fprintln(writer, "")
		ui.KeyValueSuccess(writer, "配置文件", result.ConfigPath)
		ui.KeyValueSuccess(writer, "脚本目录", result.ScriptsDir)
		fmt.Fprintln(writer, "")
	}

	ui.Info(writer, "后续可通过 %s 或 %s 浏览命令", ui.Highlight("alpen ls"), ui.Highlight("alpen ui"))
	fmt.Fprintln(writer, "")
	return nil
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
