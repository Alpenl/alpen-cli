package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/alpen/alpen-cli/internal/config"
	"github.com/spf13/cobra"
)

func loadConfig(cmd *cobra.Command, loader *config.Loader) (*config.Config, error) {
	if loader == nil {
		return nil, errors.New("配置加载器未初始化")
	}
	configPath, err := cmd.Root().PersistentFlags().GetString("config")
	if err != nil {
		return nil, err
	}
	env, err := cmd.Root().PersistentFlags().GetString("environment")
	if err != nil {
		return nil, err
	}
	cfg, err := loader.Load(configPath, env)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("未找到配置文件 %s，请先执行 'alpen init'", configPath)
		}
		return nil, err
	}
	return cfg, nil
}
