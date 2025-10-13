package config

import (
	"fmt"
	"runtime"
	"strings"
)

// Config 表示 scripts.yaml 的顶层结构
type Config struct {
	Groups map[string]Group `yaml:"groups"`
	Menus  []Menu           `yaml:"menus"`
}

// Group 对应配置中的分组信息
type Group struct {
	Description string                    `yaml:"description"`
	Scripts     map[string]ScriptTemplate `yaml:"scripts"`
}

// ScriptTemplate 表示单个脚本的定义
type ScriptTemplate struct {
	Command     string            `yaml:"command"`
	Description string            `yaml:"description"`
	Env         map[string]string `yaml:"env"`
	Platforms   []string          `yaml:"platforms"`
}

// Menu 定义顶层菜单项
type Menu struct {
	Key         string     `yaml:"key"`
	Title       string     `yaml:"title"`
	Description string     `yaml:"description"`
	Group       string     `yaml:"group"`
	Items       []MenuItem `yaml:"items"`
}

// MenuItem 描述菜单中的快捷项
type MenuItem struct {
	Key     string   `yaml:"key"`
	Label   string   `yaml:"label"`
	Script  string   `yaml:"script"`
	Args    []string `yaml:"args"`
	Aliases []string `yaml:"aliases"`
}

// Validate 对配置进行基础校验，避免出现无法执行的脚本
func (c *Config) Validate() error {
	if len(c.Groups) == 0 {
		return fmt.Errorf("groups 不能为空")
	}
	for groupName, group := range c.Groups {
		if len(group.Scripts) == 0 {
			return fmt.Errorf("组 %s 下未定义任何脚本", groupName)
		}
		for scriptName, script := range group.Scripts {
			if strings.TrimSpace(script.Command) == "" {
				return fmt.Errorf("脚本 %s/%s 未设置 command", groupName, scriptName)
			}
		}
	}
	if err := c.validateMenus(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateMenus() error {
	seen := map[string]struct{}{}
	for idx := range c.Menus {
		menu := c.Menus[idx]
		if strings.TrimSpace(menu.Key) == "" {
			return fmt.Errorf("第 %d 个菜单缺少 key", idx+1)
		}
		if _, ok := seen[menu.Key]; ok {
			return fmt.Errorf("菜单 key %s 重复", menu.Key)
		}
		seen[menu.Key] = struct{}{}
		if menu.Group == "" && len(menu.Items) == 0 {
			return fmt.Errorf("菜单 %s 必须指定 group 或至少一个 items", menu.Key)
		}
		if menu.Group != "" {
			if _, ok := c.Groups[menu.Group]; !ok {
				return fmt.Errorf("菜单 %s 引用的分组 %s 不存在", menu.Key, menu.Group)
			}
		}
		itemKeys := map[string]struct{}{}
		for itemIdx := range menu.Items {
			item := menu.Items[itemIdx]
			if strings.TrimSpace(item.Key) == "" {
				return fmt.Errorf("菜单 %s 的第 %d 个 item 缺少 key", menu.Key, itemIdx+1)
			}
			if _, ok := itemKeys[item.Key]; ok {
				return fmt.Errorf("菜单 %s 的 item key %s 重复", menu.Key, item.Key)
			}
			itemKeys[item.Key] = struct{}{}
			if strings.TrimSpace(item.Script) == "" && menu.Group == "" {
				return fmt.Errorf("菜单 %s 的 item %s 未设置 script 引用", menu.Key, item.Key)
			}
			ref := item.Script
			if strings.TrimSpace(ref) == "" {
				if menu.Group == "" {
					return fmt.Errorf("菜单 %s 的 item %s 未设置 script 引用，且菜单未绑定分组", menu.Key, item.Key)
				}
				ref = item.Key
				group := c.Groups[menu.Group]
				if _, ok := group.Scripts[item.Key]; !ok {
					return fmt.Errorf("菜单 %s 的 item %s 未显式指定 script，且在分组 %s 中找不到同名脚本", menu.Key, item.Key, menu.Group)
				}
			}
			if _, _, _, err := c.ResolveScriptRef(ref, menu.Group); err != nil {
				return fmt.Errorf("菜单 %s 的 item %s 引用脚本错误: %w", menu.Key, item.Key, err)
			}
		}
	}
	return nil
}

// FindScript 根据脚本名检索，如果存在多个同名脚本将报错提示用户精确指定
func (c *Config) FindScript(name string) (groupName string, script ScriptTemplate, err error) {
	var found []struct {
		group  string
		script ScriptTemplate
	}
	for gName, group := range c.Groups {
		if s, ok := group.Scripts[name]; ok {
			found = append(found, struct {
				group  string
				script ScriptTemplate
			}{group: gName, script: s})
		}
	}
	switch len(found) {
	case 0:
		return "", ScriptTemplate{}, fmt.Errorf("未找到脚本 %s", name)
	case 1:
		return found[0].group, found[0].script, nil
	default:
		return "", ScriptTemplate{}, fmt.Errorf("检测到多个名为 %s 的脚本，请使用 group/script 的形式", name)
	}
}

// FindScriptQualified 允许用户通过 group/script 的形式精确指定
func (c *Config) FindScriptQualified(qualified string) (groupName string, scriptName string, script ScriptTemplate, err error) {
	parts := strings.SplitN(qualified, "/", 2)
	if len(parts) != 2 {
		return "", "", ScriptTemplate{}, fmt.Errorf("脚本标识 %s 非 group/script 格式", qualified)
	}
	groupName = parts[0]
	scriptName = parts[1]
	group, ok := c.Groups[groupName]
	if !ok {
		return "", "", ScriptTemplate{}, fmt.Errorf("未找到分组 %s", groupName)
	}
	script, ok = group.Scripts[scriptName]
	if !ok {
		return "", "", ScriptTemplate{}, fmt.Errorf("分组 %s 下未找到脚本 %s", groupName, scriptName)
	}
	return groupName, scriptName, script, nil
}

// IsPlatformSupported 判断脚本是否适用当前运行平台
func (s ScriptTemplate) IsPlatformSupported() bool {
	if len(s.Platforms) == 0 {
		return true
	}
	current := runtime.GOOS
	for _, platform := range s.Platforms {
		if strings.EqualFold(platform, current) {
			return true
		}
	}
	return false
}

// FindMenu 根据 key 查找菜单
func (c *Config) FindMenu(key string) (*Menu, error) {
	for idx := range c.Menus {
		if c.Menus[idx].Key == key {
			return &c.Menus[idx], nil
		}
	}
	return nil, fmt.Errorf("未找到菜单 %s", key)
}

// ResolveScriptRef 解析菜单项中引用的脚本
func (c *Config) ResolveScriptRef(ref string, defaultGroup string) (groupName string, scriptName string, script ScriptTemplate, err error) {
	if strings.TrimSpace(ref) == "" {
		return "", "", ScriptTemplate{}, fmt.Errorf("脚本引用不能为空")
	}
	if strings.Contains(ref, "/") {
		return c.FindScriptQualified(ref)
	}
	if defaultGroup != "" {
		group, ok := c.Groups[defaultGroup]
		if !ok {
			return "", "", ScriptTemplate{}, fmt.Errorf("默认分组 %s 不存在", defaultGroup)
		}
		script, ok := group.Scripts[ref]
		if !ok {
			return "", "", ScriptTemplate{}, fmt.Errorf("在分组 %s 中未找到脚本 %s", defaultGroup, ref)
		}
		return defaultGroup, ref, script, nil
	}
	gn, scriptTpl, err := c.FindScript(ref)
	if err != nil {
		return "", "", ScriptTemplate{}, err
	}
	return gn, ref, scriptTpl, nil
}
