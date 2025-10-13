package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alpen/alpen-cli/internal/ui"
	"github.com/spf13/cobra"
)

// NewInitCommand 创建 init 子命令
func NewInitCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "初始化默认脚本配置文件",
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")
			targetDir := deps.BaseDir
			if targetDir == "" {
				var err error
				targetDir, err = os.Getwd()
				if err != nil {
					return err
				}
			}
			return initScripts(targetDir, force, cmd)
		},
	}
	cmd.Flags().Bool("force", false, "若文件已存在则覆盖")
	return cmd
}

func initScripts(baseDir string, force bool, cmd *cobra.Command) error {
	scriptsDir := filepath.Join(baseDir, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		return fmt.Errorf("创建目录 %s 失败: %w", scriptsDir, err)
	}
	targetFile := filepath.Join(scriptsDir, "scripts.yaml")

	writer := cmd.OutOrStdout()

	if _, err := os.Stat(targetFile); err == nil && !force {
		ui.Error(writer, "配置文件已存在: %s", targetFile)
		ui.Info(writer, "使用 %s 可覆盖已有文件", ui.Highlight("--force"))
		return fmt.Errorf("文件 %s 已存在，如需覆盖请使用 --force", targetFile)
	}

	if err := os.WriteFile(targetFile, []byte(defaultConfigTemplate), 0o644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	fmt.Fprintln(writer, "")
	ui.Success(writer, "已生成示例配置文件")
	fmt.Fprintf(writer, "  %s %s\n", ui.Gray("位置:"), ui.Cyan(targetFile))
	fmt.Fprintln(writer, "")
	ui.Info(writer, "可使用 %s 查看脚本列表", ui.Highlight("alpen list"))
	ui.Info(writer, "或直接运行 %s 进入交互菜单", ui.Highlight("alpen"))
	return nil
}

const defaultConfigTemplate = `# Alpen CLI 默认脚本配置示例
groups:
  build:
    description: 构建相关命令
    scripts:
      webpack-build:
        command: yarn build
        description: 打包前端资源
        env:
          NODE_ENV: production
  ops:
    description: 运维辅助命令
    scripts:
      clean-cache:
        command: rm -rf .cache
        description: 清理缓存目录
        platforms: [darwin, linux]
`
