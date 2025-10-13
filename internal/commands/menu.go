package commands

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/alpen/alpen-cli/internal/config"
	"github.com/alpen/alpen-cli/internal/executor"
	"github.com/alpen/alpen-cli/internal/ui"
	"github.com/spf13/cobra"
)

// BindRootMenu 将根命令绑定为交互式菜单入口
func BindRootMenu(root *cobra.Command, deps Dependencies) {
	root.Args = cobra.NoArgs
	root.RunE = func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig(cmd, deps.Loader)
		if err != nil {
			return err
		}
		return showTopLevelMenu(cmd, cfg, deps)
	}
}

// AttachMenus 根据配置注册菜单子命令
func AttachMenus(root *cobra.Command, deps Dependencies, cfg *config.Config) error {
	if cfg == nil {
		return nil
	}
	for _, menu := range cfg.Menus {
		if strings.TrimSpace(menu.Key) == "" {
			continue
		}
		cmd := findChildCommand(root, menu.Key)
		if cmd == nil {
			cmd = newMenuCommand(menu.Key, deps)
			root.AddCommand(cmd)
		}
		if menu.Title != "" && menu.Description != "" && menu.Title != menu.Description {
			cmd.Short = fmt.Sprintf("%s - %s", menu.Title, menu.Description)
		} else if menu.Description != "" {
			cmd.Short = menu.Description
		} else {
			cmd.Short = menu.Title
		}
		cmd.Long = fmt.Sprintf("菜单 %s：%s", menu.Key, strings.TrimSpace(menu.Description))
	}
	return nil
}

func newMenuCommand(menuKey string, deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:   menuKey,
		Short: "交互式菜单",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(cmd, deps.Loader)
			if err != nil {
				return err
			}
			menu, err := cfg.FindMenu(menuKey)
			if err != nil {
				return err
			}
			entries, err := buildMenuEntries(cfg, menu)
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				ui.Warning(cmd.OutOrStdout(), "菜单 %s 当前未配置任何脚本", menu.Key)
				return nil
			}
			if len(args) > 0 {
				entry, extraArgs, err := matchEntryByArgs(entries, args)
				if err != nil {
					return err
				}
				return executeMenuEntry(cmd, deps, entry, extraArgs)
			}
			return runMenuInteractive(cmd, menu, entries, deps)
		},
	}
	return cmd
}

func showTopLevelMenu(cmd *cobra.Command, cfg *config.Config, deps Dependencies) error {
	if len(cfg.Menus) == 0 {
		ui.Warning(cmd.OutOrStdout(), "当前配置未定义菜单，可使用 %s 查看脚本列表", ui.Cyan("alpen list"))
		return nil
	}
	reader := bufio.NewReader(cmd.InOrStdin())
	writer := cmd.OutOrStdout()

	for {
		fmt.Fprintln(writer, "")
		ui.MenuTitle(writer, "Alpen CLI 主菜单")
		for i, menu := range cfg.Menus {
			desc := strings.TrimSpace(menu.Description)
			if desc == "" {
				desc = strings.TrimSpace(menu.Title)
			}
			if desc == "" {
				desc = "未提供描述"
			}
			ui.MenuItem(writer, i+1, fmt.Sprintf("alpen %s", menu.Key), desc)
		}
		fmt.Fprintln(writer, "")
		ui.Prompt(writer, "请选择菜单 (输入序号/key 或 'q' 退出): ")
		input, err := readLine(reader)
		if err != nil {
			return nil
		}
		choice := strings.TrimSpace(input)
		if choice == "" {
			continue
		}
		lower := strings.ToLower(choice)
		if lower == "q" || lower == "quit" || lower == "exit" {
			ui.Info(writer, "已退出菜单")
			return nil
		}
		if idx, err := strconv.Atoi(choice); err == nil {
			if idx >= 1 && idx <= len(cfg.Menus) {
				menu := cfg.Menus[idx-1]
				return enterMenu(cmd, cfg, &menu, deps)
			}
			ui.Error(writer, "序号超出范围 (1-%d)，请重新选择", len(cfg.Menus))
			continue
		}
		if strings.HasPrefix(lower, "alpen ") {
			choice = strings.TrimSpace(choice[5:])
		}
		menu, err := cfg.FindMenu(choice)
		if err != nil {
			ui.Error(writer, "未找到菜单 '%s'，请输入有效的序号或 key", choice)
			continue
		}
		return enterMenu(cmd, cfg, menu, deps)
	}
}

func runMenuInteractive(cmd *cobra.Command, menu *config.Menu, entries []menuEntry, deps Dependencies) error {
	writer := cmd.OutOrStdout()
	reader := bufio.NewReader(cmd.InOrStdin())

	for {
		fmt.Fprintln(writer, "")
		ui.MenuTitle(writer, fmt.Sprintf("%s - %s", menu.Key, displayMenuTitle(menu)))
		for i, entry := range entries {
			ui.MenuItem(writer, i+1, entry.Label, entry.Description)
		}
		fmt.Fprintln(writer, "")
		ui.Prompt(writer, "请输入序号/别名 (或 'q' 退出): ")
		input, err := readLine(reader)
		if err != nil {
			return nil
		}
		choice := strings.TrimSpace(input)
		if choice == "" {
			continue
		}
		lower := strings.ToLower(choice)
		if lower == "q" || lower == "quit" || lower == "exit" {
			ui.Info(writer, "已退出菜单 %s", menu.Key)
			return nil
		}
		if idx, err := strconv.Atoi(choice); err == nil {
			if idx >= 1 && idx <= len(entries) {
				entry := entries[idx-1]
				return executeMenuEntry(cmd, deps, entry, nil)
			}
			ui.Error(writer, "序号超出范围 (1-%d)，请重新输入", len(entries))
			continue
		}
		// 尝试匹配别名
		matched := false
		for _, entry := range entries {
			if remaining, ok := entry.Match(strings.Fields(choice)); ok {
				matched = true
				return executeMenuEntry(cmd, deps, entry, remaining)
			}
		}
		if !matched {
			ui.Error(writer, "未匹配到菜单项 '%s'，请重新输入", choice)
		}
	}
}

func executeMenuEntry(cmd *cobra.Command, deps Dependencies, entry menuEntry, extraArgs []string) error {
	args := append([]string{}, entry.ExtraArgs...)
	if len(extraArgs) > 0 {
		args = append(args, extraArgs...)
	}
	req := executor.ScriptRequest{
		GroupName:  entry.GroupName,
		ScriptName: entry.ScriptName,
		Template:   entry.Template,
		ExtraArgs:  args,
		ExtraEnv:   map[string]string{},
	}

	writer := cmd.OutOrStdout()
	fmt.Fprintln(writer, "")
	ui.Executing(writer, fmt.Sprintf("%s/%s", req.GroupName, req.ScriptName))
	ui.Separator(writer)

	result, err := deps.Executor.Execute(cmd.Context(), req)
	if err != nil {
		fmt.Fprintln(writer, "")
		ui.Separator(writer)
		ui.Error(writer, "脚本执行失败")
		ui.Duration(writer, result.Duration.String())
		return err
	}

	fmt.Fprintln(writer, "")
	ui.Separator(writer)
	ui.Success(writer, "脚本执行完成")
	ui.Duration(writer, result.Duration.String())
	return nil
}

func matchEntryByArgs(entries []menuEntry, args []string) (menuEntry, []string, error) {
	for _, entry := range entries {
		if remaining, ok := entry.Match(args); ok {
			return entry, remaining, nil
		}
	}
	return menuEntry{}, nil, fmt.Errorf("未在菜单中找到匹配项: %s", strings.Join(args, " "))
}

func buildMenuEntries(cfg *config.Config, menu *config.Menu) ([]menuEntry, error) {
	result := make([]menuEntry, 0)
	if len(menu.Items) == 0 && menu.Group != "" {
		group, ok := cfg.Groups[menu.Group]
		if !ok {
			return nil, fmt.Errorf("分组 %s 不存在", menu.Group)
		}
		names := make([]string, 0, len(group.Scripts))
		for name := range group.Scripts {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			script := group.Scripts[name]
			label := script.Description
			if strings.TrimSpace(label) == "" {
				label = name
			}
			entry := menuEntry{
				Label:       label,
				Description: script.Description,
				GroupName:   menu.Group,
				ScriptName:  name,
				Template:    script,
				ExtraArgs:   nil,
				Aliases:     []aliasPattern{},
			}
			entry.addAlias(name)
			entry.addAlias(fmt.Sprintf("%s/%s", menu.Group, name))
			result = append(result, entry)
		}
		return result, nil
	}
	for _, item := range menu.Items {
		ref := item.Script
		if strings.TrimSpace(ref) == "" {
			ref = item.Key
		}
		groupName, scriptName, tmpl, err := cfg.ResolveScriptRef(ref, menu.Group)
		if err != nil {
			return nil, err
		}
		label := item.Label
		if strings.TrimSpace(label) == "" {
			label = tmpl.Description
		}
		if strings.TrimSpace(label) == "" {
			label = scriptName
		}
		entry := menuEntry{
			Label:       label,
			Description: tmpl.Description,
			GroupName:   groupName,
			ScriptName:  scriptName,
			Template:    tmpl,
			ExtraArgs:   append([]string{}, item.Args...),
			Aliases:     []aliasPattern{},
		}
		entry.addAlias(item.Key)
		for _, alias := range item.Aliases {
			entry.addAlias(alias)
		}
		entry.addAlias(scriptName)
		if menu.Group != "" {
			entry.addAlias(fmt.Sprintf("%s/%s", menu.Group, scriptName))
		} else if strings.Contains(ref, "/") {
			entry.addAlias(ref)
		}
		result = append(result, entry)
	}
	return result, nil
}

func findChildCommand(root *cobra.Command, name string) *cobra.Command {
	for _, c := range root.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}

type aliasPattern struct {
	Raw    string
	Tokens []string
}

type menuEntry struct {
	Label       string
	Description string
	GroupName   string
	ScriptName  string
	Template    config.ScriptTemplate
	ExtraArgs   []string
	Aliases     []aliasPattern
}

func (e *menuEntry) addAlias(raw string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return
	}
	tokens := strings.Fields(raw)
	e.Aliases = append(e.Aliases, aliasPattern{
		Raw:    raw,
		Tokens: tokens,
	})
}

// Match 判断传入参数是否匹配当前菜单项，并返回剩余参数
func (e menuEntry) Match(args []string) ([]string, bool) {
	for _, alias := range e.Aliases {
		if len(alias.Tokens) == 0 {
			continue
		}
		if len(args) < len(alias.Tokens) {
			continue
		}
		match := true
		for i, token := range alias.Tokens {
			if args[i] != token {
				match = false
				break
			}
		}
		if match {
			return args[len(alias.Tokens):], true
		}
	}
	return nil, false
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			if len(line) == 0 {
				return "", err
			}
			return line, nil
		}
		return "", err
	}
	return line, nil
}

func enterMenu(cmd *cobra.Command, cfg *config.Config, menu *config.Menu, deps Dependencies) error {
	entries, err := buildMenuEntries(cfg, menu)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		ui.Warning(cmd.OutOrStdout(), "菜单 %s 当前未配置任何脚本", menu.Key)
		return nil
	}
	return runMenuInteractive(cmd, menu, entries, deps)
}

func displayMenuTitle(menu *config.Menu) string {
	title := strings.TrimSpace(menu.Title)
	if title == "" {
		title = strings.TrimSpace(menu.Description)
	}
	if title == "" {
		title = "未命名菜单"
	}
	return title
}
