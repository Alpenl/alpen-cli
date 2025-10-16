package config

import (
	"fmt"
	"sort"
	"strings"
)

// Config 表示 demo.yaml 的顶层结构
type Config struct {
	Commands    map[string]CommandSpec `yaml:"commands"`
	Diagnostics []Diagnostic           `yaml:"-"`
}

// CommandSpec 定义一级命令的元数据
type CommandSpec struct {
	Alias       string                `yaml:"alias"`
	Description string                `yaml:"description"`
	Command     string                `yaml:"command"`
	Actions     map[string]ActionSpec `yaml:"actions"`
	Origin      SourceInfo            `yaml:"-"`
}

// ActionSpec 定义子命令的元数据
type ActionSpec struct {
	Alias       string     `yaml:"alias"`
	Description string     `yaml:"description"`
	Command     string     `yaml:"command"`
	Origin      SourceInfo `yaml:"-"`
}

// Validate 对配置进行基础校验，保证命令结构可执行
func (c *Config) Validate() error {
	if len(c.Commands) == 0 {
		return fmt.Errorf("commands 不能为空")
	}
	aliasUsage := map[string]string{}
	for name, spec := range c.Commands {
		if err := validateIdentifier("命令名称", name); err != nil {
			return err
		}
		if alias := strings.TrimSpace(spec.Alias); alias != "" {
			if err := validateIdentifier(fmt.Sprintf("命令 %s 的别名", name), alias); err != nil {
				return err
			}
			if owner, exists := aliasUsage[alias]; exists {
				return fmt.Errorf("命令 %s 的别名 %s 与命令 %s 冲突", name, alias, owner)
			}
			aliasUsage[alias] = name
		}
		if err := validateCommandSpec(name, spec); err != nil {
			return err
		}
	}
	return nil
}

func validateCommandSpec(name string, spec CommandSpec) error {
	if spec.Actions == nil {
		spec.Actions = map[string]ActionSpec{}
	}
	if strings.TrimSpace(spec.Command) == "" && len(spec.Actions) == 0 {
		return fmt.Errorf("命令 %s 需要提供默认 command 或至少一个 action", name)
	}
	actionAliases := map[string]string{}
	for actionName, action := range spec.Actions {
		if err := validateIdentifier(fmt.Sprintf("命令 %s 的子命令名称", name), actionName); err != nil {
			return err
		}
		if strings.TrimSpace(action.Command) == "" {
			return fmt.Errorf("命令 %s 的子命令 %s 缺少 command", name, actionName)
		}
		if alias := strings.TrimSpace(action.Alias); alias != "" {
			if err := validateIdentifier(fmt.Sprintf("命令 %s 的子命令 %s 的别名", name, actionName), alias); err != nil {
				return err
			}
			if owner, exists := actionAliases[alias]; exists {
				return fmt.Errorf("命令 %s 的子命令别名 %s 同时被 %s 与 %s 使用", name, alias, owner, actionName)
			}
			actionAliases[alias] = actionName
		}
	}
	return nil
}

// SortedCommandNames 返回排序后的命令名称，便于稳定输出
func (c *Config) SortedCommandNames() []string {
	names := make([]string, 0, len(c.Commands))
	for name := range c.Commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// SortedActionNames 返回排序后的子命令名称
func (c *CommandSpec) SortedActionNames() []string {
	if len(c.Actions) == 0 {
		return nil
	}
	names := make([]string, 0, len(c.Actions))
	for name := range c.Actions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func validateIdentifier(label, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("%s不能为空", label)
	}
	if trimmed != value {
		return fmt.Errorf("%s %q 含有首尾空白字符", label, value)
	}
	if strings.ContainsAny(trimmed, " \t\n\r") {
		return fmt.Errorf("%s %s 不应包含空白字符", label, trimmed)
	}
	return nil
}
