package commands

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/spf13/cobra"

	"github.com/alpen/alpen-cli/internal/bootstrap"
	"github.com/alpen/alpen-cli/internal/config"
	"github.com/alpen/alpen-cli/internal/ui"
)

const defaultConfigRelativePath = ".alpen/demo.yaml"

// NewEnvCommand 创建 env 子命令，提供配置文件选择界面
func NewEnvCommand(deps Dependencies) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "env",
		Aliases:       []string{"environment"},
		Short:         "选择并激活配置文件",
		Long:          "列出 ~/.alpen/config 目录中的配置文件，选择后写入状态目录，之后执行的命令都会使用该配置。",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runEnvSelector(cmd, deps)
		},
	}
	cmd.Flags().Bool("reset", false, "重建 ~/.alpen/config 下的示例内容")
	return cmd
}

func runEnvSelector(cmd *cobra.Command, deps Dependencies) error {
	reset, _ := cmd.Flags().GetBool("reset")

	if reset {
		return handleEnvReset(cmd)
	}

	ctx, err := buildEnvSelectionContext(cmd, deps)
	if err != nil {
		return err
	}
	if ctx == nil || len(ctx.options) == 0 {
		renderNoConfigHint(cmd.OutOrStdout())
		return nil
	}

	writer := cmd.OutOrStdout()
	fmt.Fprintln(writer, "")
	if strings.TrimSpace(ctx.activePath) != "" {
		display := filepath.Base(ctx.activePath)
		if strings.TrimSpace(display) == "" || display == "." {
			display = ctx.activePath
		}
		ui.KeyValue(writer, "当前激活配置", display)
	}
	fmt.Fprintln(writer, "")

	choice, err := promptConfigSelection(ctx.groups, ctx.activePath)
	if err != nil {
		if errors.Is(err, io.EOF) || err.Error() == "interrupt" {
			fmt.Fprintln(writer, "")
			fmt.Fprintln(writer, "")
			fmt.Fprintln(writer, ui.Yellow("    已取消"))
			return nil
		}
		return err
	}
	if err := persistEnvSelection(choice); err != nil {
		return err
	}

	envName, _ := cmd.Root().PersistentFlags().GetString("environment")
	renderSelectionResult(writer, choice, deps.Loader, envName)
	return nil
}

type envSelectionContext struct {
	options    []configCandidate
	groups     []configGroup
	activePath string
}

type configGroup struct {
	Title   string
	Options []configCandidate
}

func handleEnvReset(cmd *cobra.Command) error {
	writer := cmd.OutOrStdout()

	result, err := bootstrap.EnsureHomeAssets(true)
	if err != nil {
		return err
	}
	if err := bootstrap.EnsureHomeReadme(filepath.Dir(result.ConfigPath), true); err != nil {
		ui.Warning(writer, "写入 README 失败: %v", err)
	}
	if err := config.SaveActiveConfigPath(result.ConfigPath); err != nil {
		return err
	}

	ui.Success(writer, "已重置默认配置路径")
	return nil
}

func buildEnvSelectionContext(cmd *cobra.Command, deps Dependencies) (*envSelectionContext, error) {
	configFlagPath, err := currentConfigPath(cmd, deps)
	if err != nil {
		return nil, err
	}

	configDir, err := config.ConfigDir()
	if err != nil {
		return nil, err
	}

	searchDirs, defaultPath := collectSearchDirs(configDir)

	candidates, err := listConfigFiles(searchDirs)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return &envSelectionContext{}, nil
	}

	activePath, err := config.LoadActiveConfigPath()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(activePath) == "" {
		activePath = defaultPath
	}
	if strings.TrimSpace(activePath) == "" {
		activePath = configFlagPath
	}
	if strings.TrimSpace(activePath) == "" || !dirInsideAllowed(filepath.Dir(activePath), searchDirs) {
		activePath = defaultPath
	}
	if strings.TrimSpace(activePath) == "" && len(candidates) > 0 {
		activePath = candidates[0].AbsolutePath
	}

	options, groups := buildConfigGroups(candidates)
	return &envSelectionContext{
		options:    options,
		groups:     groups,
		activePath: activePath,
	}, nil
}

func promptConfigSelection(groups []configGroup, activePath string) (configCandidate, error) {
	var options []string
	var mapping []configCandidate
	var metas []selectOptionTemplateMeta
	defaultIndex := -1
	previousGroup := ""
	for _, group := range groups {
		for _, option := range group.Options {
			isActive := samePath(activePath, option.AbsolutePath)
			if isActive {
				defaultIndex = len(options)
			}
			isNewGroup := previousGroup != group.Title
			metaGroup := group.Title
			first := isNewGroup
			gap := isNewGroup && len(options) > 0
			if strings.TrimSpace(metaGroup) == "" {
				metaGroup = ""
				first = false
				gap = false
			}
			meta := selectOptionTemplateMeta{
				Group:     metaGroup,
				Name:      option.DisplayName,
				Path:      option.AbsolutePath,
				Active:    isActive,
				First:     first,
				GapBefore: gap,
			}
			previousGroup = group.Title
			metas = append(metas, meta)
			options = append(options, option.DisplayName)
			mapping = append(mapping, option)
		}
	}
	if len(mapping) == 0 {
		return configCandidate{}, errors.New("无可用配置可供选择")
	}
	ensureEnvSelectTemplate()
	cleanupMeta := setSelectOptionMeta(metas)
	defer cleanupMeta()

	prompt := &survey.Select{
		Message:  "选择配置文件 (↑/↓ 导航 | / 搜索 | Enter 确认)",
		Options:  options,
		PageSize: minInt(15, len(options)),
		Filter:   buildEnvFilter(mapping, metas),
	}
	if defaultIndex >= 0 {
		prompt.Default = defaultIndex
	}
	var selected int
	if err := survey.AskOne(prompt, &selected); err != nil {
		return configCandidate{}, err
	}
	if selected < 0 || selected >= len(mapping) {
		return configCandidate{}, fmt.Errorf("选择索引超出范围")
	}
	return mapping[selected], nil
}

func buildEnvFilter(options []configCandidate, metas []selectOptionTemplateMeta) func(filter string, value string, index int) bool {
	return func(filter string, value string, index int) bool {
		trimmed := strings.TrimSpace(filter)
		if trimmed == "" {
			return true
		}
		if index < 0 || index >= len(options) {
			return false
		}

		candidate := options[index]
		prefixTokens := collectEnvPrefixTokens(candidate, metas, index)
		searchTokens := collectEnvSearchTokens(candidate, metas, index)
		lowerValue := strings.ToLower(value)

		if strings.HasPrefix(trimmed, "/") {
			prefix := strings.TrimLeft(trimmed, "/")
			prefix = strings.ToLower(strings.TrimSpace(prefix))
			if prefix == "" {
				return true
			}
			for _, token := range prefixTokens {
				if strings.HasPrefix(token, prefix) {
					return true
				}
			}
			return false
		}

		needle := strings.ToLower(trimmed)
		if strings.Contains(lowerValue, needle) {
			return true
		}
		for _, token := range searchTokens {
			if strings.Contains(token, needle) {
				return true
			}
		}
		return false
	}
}

func collectEnvSearchTokens(candidate configCandidate, metas []selectOptionTemplateMeta, index int) []string {
	addToken := func(tokens []string, value string) []string {
		value = strings.TrimSpace(value)
		if value == "" {
			return tokens
		}
		return append(tokens, strings.ToLower(value))
	}

	tokens := make([]string, 0, 12)
	tokens = addToken(tokens, candidate.DisplayName)
	tokens = addToken(tokens, candidate.AbsolutePath)
	tokens = addToken(tokens, filepath.Base(candidate.AbsolutePath))

	dir := filepath.Dir(candidate.AbsolutePath)
	if dir != "" && dir != "." {
		tokens = addToken(tokens, dir)
	}

	for _, part := range strings.FieldsFunc(filepath.ToSlash(candidate.AbsolutePath), func(r rune) bool { return r == '/' }) {
		tokens = addToken(tokens, part)
	}

	if index >= 0 && index < len(metas) {
		meta := metas[index]
		tokens = addToken(tokens, meta.Group)
		tokens = addToken(tokens, meta.Name)
		tokens = addToken(tokens, meta.Path)
	}
	return tokens
}

func collectEnvPrefixTokens(candidate configCandidate, metas []selectOptionTemplateMeta, index int) []string {
	addToken := func(tokens []string, value string) []string {
		value = strings.TrimSpace(value)
		if value == "" {
			return tokens
		}
		return append(tokens, strings.ToLower(value))
	}

	tokens := make([]string, 0, 6)
	tokens = addToken(tokens, candidate.DisplayName)
	tokens = addToken(tokens, filepath.Base(candidate.AbsolutePath))

	if index >= 0 && index < len(metas) {
		meta := metas[index]
		tokens = addToken(tokens, meta.Name)
		tokens = addToken(tokens, meta.Group)
	}
	return tokens
}

func persistEnvSelection(chosen configCandidate) error {
	return config.SaveActiveConfigPath(chosen.AbsolutePath)
}

func renderSelectionResult(writer io.Writer, chosen configCandidate, loader *config.Loader, envName string) {
	ui.KeyValueSuccess(writer, "已激活配置", filepath.Base(chosen.AbsolutePath))
	if loader == nil {
		return
	}
	cfg, err := loader.Load(chosen.AbsolutePath, envName)
	if err != nil {
		fmt.Fprintln(writer, "")
		ui.Warning(writer, "解析配置时出现问题: %v", err)
		return
	}
	renderDiagnostics(writer, cfg.Diagnostics)
}

func renderNoConfigHint(writer io.Writer) {
	ui.Warning(writer, "未在目录中找到可用的配置文件")
	ui.Info(writer, "可执行 %s 创建默认配置", ui.Highlight("alpen init"))
}

func currentConfigPath(cmd *cobra.Command, deps Dependencies) (string, error) {
	root := cmd.Root()
	configFlag, err := root.PersistentFlags().GetString("config")
	if err != nil {
		return "", err
	}
	configFlag = config.ExpandPath(configFlag)
	if strings.TrimSpace(configFlag) == "" {
		configFlag = defaultConfigRelativePath
	}
	if filepath.IsAbs(configFlag) {
		return configFlag, nil
	}
	if deps.BaseDir == "" {
		return configFlag, nil
	}
	return filepath.Join(deps.BaseDir, configFlag), nil
}

type configCandidate struct {
	DisplayName  string
	AbsolutePath string
	Group        string
}

func listConfigFiles(dirs []string) ([]configCandidate, error) {
	var candidates []configCandidate
	seen := map[string]struct{}{}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			abs := filepath.Join(dir, entry.Name())
			if entry.IsDir() {
				if !isConfigDirectory(entry) {
					continue
				}
			} else if !isConfigFile(entry) {
				continue
			}
			if _, ok := seen[abs]; ok {
				continue
			}
			seen[abs] = struct{}{}
			display := entry.Name()
			candidates = append(candidates, configCandidate{
				DisplayName:  display,
				AbsolutePath: abs,
			})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].DisplayName < candidates[j].DisplayName
	})
	return candidates, nil
}

func isConfigFile(entry fs.DirEntry) bool {
	name := strings.ToLower(entry.Name())
	return strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml")
}

func isConfigDirectory(entry fs.DirEntry) bool {
	return strings.HasSuffix(strings.ToLower(entry.Name()), ".conf")
}

func buildConfigGroups(candidates []configCandidate) ([]configCandidate, []configGroup) {
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].DisplayName < candidates[j].DisplayName
	})
	options := make([]configCandidate, len(candidates))
	copy(options, candidates)
	for i := range options {
		options[i].Group = ""
	}
	groups := []configGroup{
		{Title: "", Options: options},
	}
	return options, groups
}

func samePath(a, b string) bool {
	cleanA := filepath.Clean(a)
	cleanB := filepath.Clean(b)
	return cleanA == cleanB
}

func collectSearchDirs(configDir string) ([]string, string) {
	cleanDir := filepath.Clean(strings.TrimSpace(configDir))
	var dirs []string
	if cleanDir != "" {
		dirs = append(dirs, cleanDir)
	}

	defaultPath := ""
	if cleanDir != "" {
		defaultPath = filepath.Join(cleanDir, "demo.yaml")
	} else if len(dirs) > 0 {
		defaultPath = filepath.Join(dirs[0], "demo.yaml")
	}
	return dirs, defaultPath
}

func dirInsideAllowed(target string, allowed []string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	target = filepath.Clean(target)
	for _, dir := range allowed {
		if dir == "" {
			continue
		}
		if insideDir(target, dir) {
			return true
		}
	}
	return false
}

func insideDir(path, dir string) bool {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return false
	}
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	return rel == "." || !strings.HasPrefix(rel, "..")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
