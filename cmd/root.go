package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/alpen/alpen-cli/internal/commands"
	"github.com/alpen/alpen-cli/internal/config"
	"github.com/alpen/alpen-cli/internal/executor"
	"github.com/alpen/alpen-cli/internal/plugins"
	"github.com/spf13/cobra"
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
}

// Execute 是主程序入口
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	baseDir, err := os.Getwd()
	if err != nil {
		panic(fmt.Errorf("获取工作目录失败: %w", err))
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

	rootCmd.PersistentFlags().StringP("config", "c", "scripts/scripts.yaml", "指定脚本配置文件路径")
	rootCmd.PersistentFlags().StringP("environment", "e", "", "指定环境名称，用于加载环境差异配置")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if !requiresConfig(cmd) {
			return nil
		}
		if _, err := os.Stat(filepath.Join(baseDir, "scripts")); os.IsNotExist(err) {
			return fmt.Errorf("未找到 scripts 目录，请先执行 'alpen init' 或创建配置")
		}
		return nil
	}
	rootCmd.AddCommand(newVersionCmd())
	commands.Register(rootCmd, deps)

	bootstrapMenus(rootCmd, deps, loader, logger)

	cobra.OnInitialize(func() {
		configPath, err := rootCmd.PersistentFlags().GetString("config")
		if err != nil {
			logger.Printf("读取配置路径失败: %v", err)
			return
		}
		envName, err := rootCmd.PersistentFlags().GetString("environment")
		if err != nil {
			logger.Printf("读取 environment 标志失败: %v", err)
			return
		}
		cfg, err := loader.Load(configPath, envName)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return
			}
			logger.Printf("加载菜单配置失败: %v", err)
			return
		}
		if err := commands.AttachMenus(rootCmd, deps, cfg); err != nil {
			logger.Printf("注册菜单失败: %v", err)
		}
	})
}

// newVersionCmd 输出版本信息
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "查看当前版本信息",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "Alpen CLI %s (commit: %s, build date: %s)\n", version, commit, date)
		},
	}
}

func requiresConfig(cmd *cobra.Command) bool {
	if cmd == nil {
		return true
	}
	switch cmd.Name() {
	case "init", "version", "help":
		return false
	}
	if cmd.HasParent() && cmd.Parent() != nil && cmd.Parent().Name() == "alpen" {
		// 对子命令做同样判断
		switch cmd.Name() {
		case "init", "version", "help":
			return false
		}
	}
	return true
}

func bootstrapMenus(root *cobra.Command, deps commands.Dependencies, loader *config.Loader, logger *log.Logger) {
	configPath, envName := detectInitialFlags(os.Args[1:])
	if configPath == "" {
		configPath = "scripts/scripts.yaml"
	}
	cfg, err := loader.Load(configPath, envName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		logger.Printf("初始化菜单时加载配置失败: %v", err)
		return
	}
	if err := commands.AttachMenus(root, deps, cfg); err != nil {
		logger.Printf("初始化注册菜单失败: %v", err)
	}
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
		case arg == "-e":
			if i+1 < stop {
				environment = args[i+1]
				i++
			}
		case strings.HasPrefix(arg, "-e="):
			environment = strings.TrimPrefix(arg, "-e=")
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
