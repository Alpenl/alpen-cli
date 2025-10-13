package commands

import (
	"log"
	"os"

	"github.com/alpen/alpen-cli/internal/config"
	"github.com/alpen/alpen-cli/internal/executor"
	"github.com/alpen/alpen-cli/internal/plugins"
	"github.com/spf13/cobra"
)

// Dependencies 用于在命令之间共享核心组件
type Dependencies struct {
	Loader   *config.Loader
	Executor *executor.Executor
	Registry *plugins.Registry
	Logger   *log.Logger
	BaseDir  string
}

// Register 将所有子命令挂载到根命令
func Register(root *cobra.Command, deps Dependencies) {
	if deps.Logger == nil {
		deps.Logger = log.New(os.Stdout, "[alpen] ", log.LstdFlags)
	}
	root.AddCommand(NewListCommand(deps))
	root.AddCommand(NewRunCommand(deps))
	root.AddCommand(NewInitCommand(deps))
	BindRootMenu(root, deps)
}
