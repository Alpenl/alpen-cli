package cmd

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/alpen/alpen-cli/internal/commands"
	"github.com/alpen/alpen-cli/internal/config"
	"github.com/alpen/alpen-cli/internal/executor"
	"github.com/alpen/alpen-cli/internal/plugins"
	"github.com/alpen/alpen-cli/internal/ui"
)

var (
	version = "0.1.0" // 后续可通过编译时 ldflags 注入
	commit  = "dev"
	date    = "unknown"
)

// rootCmd 负责定义 CLI 根命令
var rootCmd = &cobra.Command{
	Use:   "alpen",
	Short: "Alpen CLI - 团队脚本统一入口",
	Long:  "Alpen CLI 提供脚本统一管理与执行能力，支持按配置驱动的脚本维护方式。",
	RunE: func(cmd *cobra.Command, args []string) error {
		showVersion, err := cmd.Flags().GetBool("version")
		if err != nil {
			return err
		}
		if showVersion {
			printVersion(cmd.OutOrStdout())
			return nil
		}
		return showWelcome(cmd)
	},
}

// Execute 是主程序入口
func Execute() error {
	preprocessArgs()
	start := time.Now()
	err := rootCmd.Execute()
	if err != nil {
		if !commands.IsReportedError(err) {
			writer := rootCmd.ErrOrStderr()
			displayName := strings.Join(os.Args[1:], " ")
			displayName = strings.TrimSpace(displayName)
			if displayName == "" {
				displayName = rootCmd.Name()
			}
			ui.BeginExecution(writer, displayName)
			ui.EndExecution(writer)
			ui.ExecutionSummary(writer, false, time.Since(start), translateRootError(err))
		}
		return err
	}
	return nil
}

func init() {
	baseDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取工作目录失败，已回退为当前目录: %v\n", err)
		baseDir = "."
	}
	if home, err := config.ResolveHomeDir(); err == nil {
		_ = os.Setenv("ALPEN_HOME", home)
	}
	loader := config.NewLoader(baseDir)
	registry := plugins.NewRegistry()
	logger := log.New(os.Stdout, "[alpen] ", log.LstdFlags)
	exec := executor.NewExecutor(registry, logger)
	deps := commands.Dependencies{
		Loader:   loader,
		Executor: exec,
		Registry: registry,
		Logger:   logger,
		BaseDir:  baseDir,
	}

	defaultConfigPath, err := config.NormalizeConfigPath("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "计算默认配置路径失败: %v\n", err)
		defaultConfigPath = "."
	}
	rootCmd.PersistentFlags().StringP("config", "c", defaultConfigPath, "指定命令配置文件路径（仅限 ~/.alpen 下的文件）")
	rootCmd.PersistentFlags().String("environment", "", "指定环境名称，用于加载环境差异配置")
	rootCmd.PersistentFlags().BoolP("version", "v", false, "查看当前版本信息")
	rootCmd.SilenceErrors = true
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		path, err := cmd.Root().PersistentFlags().GetString("config")
		if err != nil {
			return err
		}
		normalized, err := config.NormalizeConfigPath(path)
		if err != nil {
			return err
		}
		if normalized != path {
			if setErr := cmd.Root().PersistentFlags().Set("config", normalized); setErr != nil {
				return setErr
			}
		}
		return nil
	}
	rootCmd.AddCommand(newVersionCmd())
	commands.Register(rootCmd, deps)

	loaded, configPathUsed, loadErr := bootstrapCommands(rootCmd, deps, loader, logger)
	if loadErr != nil {
		logger.Printf("初始化加载配置失败: %v", loadErr)
	}
	if !loaded {
		originalRunE := rootCmd.RunE
		rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
			showVersion, err := cmd.Flags().GetBool("version")
			if err != nil {
				return err
			}
			if showVersion {
				return originalRunE(cmd, args)
			}

			writer := cmd.OutOrStdout()
			hintPath := configPathUsed
			if hintPath == "" {
				if computed, err := config.NormalizeConfigPath(""); err == nil {
					hintPath = computed
				}
			}
			ui.Warning(writer, "未检测到命令配置文件 %s", ui.Highlight(hintPath))
			ui.Info(writer, "可执行 %s 生成默认配置示例", ui.Highlight("alpen init"))
			ui.Info(writer, "如需切换其它配置，请确保文件位于 ~/.alpen 内并使用 %s 指定", ui.Highlight("--config"))
			fmt.Fprintln(writer, "")
			return cmd.Help()
		}
	}
}

// newVersionCmd 输出版本信息
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "查看当前版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			printVersion(cmd.OutOrStdout())
		},
	}
}

func printVersion(writer io.Writer) {
	fmt.Fprintf(writer, "Alpen CLI %s (commit: %s, build date: %s)\n", version, commit, date)
}

func bootstrapCommands(root *cobra.Command, deps commands.Dependencies, loader *config.Loader, _ *log.Logger) (bool, string, error) {
	configPath, envName := detectInitialFlags(os.Args[1:])
	if configPath == "" {
		if active, err := config.LoadActiveConfigPath(); err == nil && strings.TrimSpace(active) != "" {
			configPath = active
		}
	}
	normalized, err := config.NormalizeConfigPath(configPath)
	if err != nil {
		if strings.TrimSpace(configPath) != "" {
			fmt.Fprintf(os.Stderr, "配置路径无效，将回退为默认值: %v\n", err)
		}
		normalized, err = config.NormalizeConfigPath("")
		if err != nil {
			return false, "", err
		}
	}
	configPath = normalized
	if err := root.PersistentFlags().Set("config", configPath); err != nil {
		return false, configPath, err
	}
	cfg, err := loader.Load(configPath, envName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, configPath, nil
		}
		return false, configPath, err
	}
	if err := commands.RegisterDynamicCommands(root, deps, cfg); err != nil {
		return false, configPath, err
	}
	return true, configPath, nil
}

func detectInitialFlags(args []string) (configPath string, environment string) {
	stop := len(args)
	for i, arg := range args {
		if arg == "--" {
			stop = i
			break
		}
	}
	for i := 0; i < stop; i++ {
		arg := args[i]
		switch {
		case arg == "-c":
			if i+1 < stop {
				configPath = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "-c="):
			configPath = strings.TrimPrefix(arg, "-c=")
		case arg == "--config":
			if i+1 < stop {
				configPath = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--config="):
			configPath = strings.TrimPrefix(arg, "--config=")
		case arg == "--environment":
			if i+1 < stop {
				environment = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "--environment="):
			environment = strings.TrimPrefix(arg, "--environment=")
		}
	}
	return configPath, environment
}

// showWelcome 显示美化的欢迎页面
func showWelcome(cmd *cobra.Command) error {
	w := cmd.OutOrStdout()

	commands := []ui.CommandInfo{
		{Name: "env", Description: "选择并激活配置文件"},
		{Name: "ls", Description: "快速查看配置中的命令列表"},
		{Name: "ui", Description: "交互式命令导航 (推荐)"},
		{Name: "init", Description: "初始化默认配置文件"},
		{Name: "version", Description: "查看版本信息"},
	}

	ui.ShowWelcome(w, commands)
	return nil
}

// preprocessArgs 在执行前将特殊写法转换为 CLI 可识别的命令
func preprocessArgs() {
	if len(os.Args) < 2 {
		return
	}
	if os.Args[1] == "-e" && len(os.Args) == 2 {
		os.Args[1] = "env"
	}
}

func translateRootError(err error) error {
	if err == nil {
		return nil
	}

	msg := strings.TrimSpace(err.Error())

	const unknownPrefix = "unknown command "
	if strings.HasPrefix(msg, unknownPrefix) {
		name := extractQuotedSegment(msg)
		if name == "" && len(os.Args) > 1 {
			name = os.Args[1]
		}
		name = strings.TrimSpace(name)
		var builder strings.Builder
		if name != "" {
			builder.WriteString(fmt.Sprintf("未识别的命令：%s", name))
		} else {
			builder.WriteString("未识别的命令")
		}
		suggestions := rootCmd.SuggestionsFor(name)
		if len(suggestions) > 0 {
			builder.WriteString("\n  建议尝试：")
			builder.WriteString(strings.Join(suggestions, "、"))
		} else {
			builder.WriteString("\n  建议执行：alpen ls 查看可用命令")
		}
		return errors.New(builder.String())
	}

	return err
}

func extractQuotedSegment(text string) string {
	start := strings.IndexRune(text, '"')
	if start == -1 {
		return ""
	}
	rest := text[start+1:]
	end := strings.IndexRune(rest, '"')
	if end == -1 {
		return ""
	}
	return rest[:end]
}
