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
			useLocal, _ := cmd.Flags().GetBool("local")
			if useLocal {
				return initLocalConfig(deps.BaseDir, force, cmd)
			}
			return initGlobalConfig(force, cmd)
		},
	}
	cmd.Flags().Bool("force", false, "若文件已存在则覆盖")
	cmd.Flags().BoolP("local", "l", false, "在当前项目下生成示例配置文件")
	return cmd
}

func initGlobalConfig(force bool, cmd *cobra.Command) error {
	writer := cmd.OutOrStdout()

	globalCfg, err := config.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("读取 global.yaml 失败: %w", err)
	}

	configPathHint := strings.TrimSpace(globalCfg.DefaultConfigPath)
	if configPathHint == "" {
		home, err := config.ResolveHomeDir()
		if err != nil {
			return err
		}
		configPathHint = filepath.Join(home, "config", "demo.yaml")
	}
	configPathHint = config.ExpandPath(configPathHint)
	alreadyExists := !force && fileExists(configPathHint)

	result, err := bootstrap.EnsureGlobalAssets(globalCfg, force)
	if err != nil {
		return err
	}

	if err := bootstrap.EnsureGlobalReadme(filepath.Dir(result.ConfigPath), force); err != nil {
		ui.Warning(writer, "写入 README 失败: %v", err)
	}

	globalCfg.DefaultConfigPath = result.ConfigPath
	globalCfg.ScriptsRoot = result.ScriptsDir
	configDir := filepath.Dir(result.ConfigPath)
	globalCfg.SearchPaths = bootstrap.UniqueStrings(append([]string{configDir}, globalCfg.SearchPaths...))

	if err := bootstrap.PersistGlobalConfig(globalCfg, force); err != nil {
		return fmt.Errorf("写入 global.yaml 失败: %w", err)
	}

	if err := config.SaveActiveConfigPath(result.ConfigPath); err != nil {
		ui.Warning(writer, "无法记录激活配置: %v", err)
	}

	fmt.Fprintln(writer, "")
	if alreadyExists {
		ui.Info(writer, "检测到全局配置已存在，未执行覆盖操作")
	} else {
		ui.Success(writer, "已在全局目录生成示例配置")
		fmt.Fprintf(writer, "  %s %s\n", ui.Gray("配置文件:"), ui.Cyan(result.ConfigPath))
		fmt.Fprintf(writer, "  %s %s\n", ui.Gray("脚本目录:"), ui.Cyan(result.ScriptsDir))
		fmt.Fprintln(writer, "")
	}
	if alreadyExists {
		fmt.Fprintf(writer, "  %s %s\n", ui.Gray("配置文件:"), ui.Cyan(result.ConfigPath))
		fmt.Fprintf(writer, "  %s %s\n", ui.Gray("脚本目录:"), ui.Cyan(result.ScriptsDir))
		fmt.Fprintln(writer, "")
		ui.Info(writer, "如需覆盖示例配置，可使用 %s", ui.Highlight("alpen init --force"))
	}
	ui.Info(writer, "后续可通过 %s 或 %s 浏览命令", ui.Highlight("alpen ls"), ui.Highlight("alpen ui"))
	ui.Info(writer, "在项目内生成配置，可使用 %s 或 %s", ui.Highlight("alpen init -local"), ui.Highlight("alpen init -l"))
	return nil
}

func initLocalConfig(baseDir string, force bool, cmd *cobra.Command) error {
	writer := cmd.OutOrStdout()

	configPath := localConfigPath(baseDir)
	alreadyExists := !force && fileExists(configPath)

	targetFile, err := bootstrap.EnsureLocalAssets(baseDir, force)
	if err != nil {
		return err
	}

	projectDir := filepath.Dir(targetFile)
	existingActive := ""
	if active, loadErr := config.LoadProjectActiveConfigPath(projectDir); loadErr == nil {
		existingActive = strings.TrimSpace(active)
	} else if !os.IsNotExist(loadErr) {
		ui.Warning(writer, "读取项目激活配置失败: %v", loadErr)
	}
	if force || strings.TrimSpace(existingActive) == "" {
		if saveErr := config.SaveProjectActiveConfigPath(projectDir, targetFile); saveErr != nil {
			ui.Warning(writer, "记录项目激活配置失败: %v", saveErr)
		}
	}

	fmt.Fprintln(writer, "")
	if alreadyExists {
		ui.Info(writer, "检测到当前项目已完成初始化")
	} else {
		ui.Success(writer, "已在当前项目生成示例配置")
	}
	fmt.Fprintf(writer, "  %s %s\n", ui.Gray("位置:"), ui.Cyan(targetFile))
	fmt.Fprintln(writer, "")
	if alreadyExists {
		ui.Info(writer, "如需覆盖示例配置，可使用 %s", ui.Highlight("alpen init -local --force"))
	}
	ui.Info(writer, "可使用 %s 查看命令结构", ui.Highlight("alpen ls"))
	ui.Info(writer, "可参考示例结构补充自定义命令")
	return nil
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func localConfigPath(baseDir string) string {
	targetDir := strings.TrimSpace(baseDir)
	if targetDir == "" {
		cwd, err := os.Getwd()
		if err == nil {
			targetDir = cwd
		}
	}
	if targetDir == "" {
		return ""
	}
	return filepath.Join(targetDir, ".alpen", "demo.yaml")
}
