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
		Long:          "列出可用的命令配置文件，选择后写入全局状态，之后执行的命令都会使用该配置。",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runEnvSelector(cmd, deps)
		},
	}
	cmd.Flags().Bool("reset", false, "重置全局配置路径到默认值")
	return cmd
}

func runEnvSelector(cmd *cobra.Command, deps Dependencies) error {
	reset, _ := cmd.Flags().GetBool("reset")

	globalCfg, err := config.LoadGlobalConfig()
	if err != nil {
		return fmt.Errorf("读取全局配置失败: %w", err)
	}
	if reset {
		return handleEnvReset(cmd, globalCfg)
	}

	if _, err := bootstrap.EnsureGlobalAssets(globalCfg, false); err != nil {
		return err
	}

	ctx, err := buildEnvSelectionContext(cmd, deps, globalCfg)
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
		fmt.Fprintf(writer, "  %s %s\n", ui.Gray("当前激活配置:"), ui.Cyan(ctx.activePath))
	}
	fmt.Fprintln(writer, "")
	fmt.Fprintln(writer, "")

	choice, err := promptConfigSelection(ctx.groups, ctx.activePath)
	if err != nil {
		if errors.Is(err, io.EOF) || err.Error() == "interrupt" {
			ui.Info(writer, "已取消配置切换")
			return nil
		}
		return err
	}
	if err := persistEnvSelection(globalCfg, ctx.projectDir, ctx.globalDir, choice); err != nil {
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
	projectDir string
	globalDir  string
}

type configGroup struct {
	Title   string
	Options []configCandidate
}

func handleEnvReset(cmd *cobra.Command, current *config.GlobalConfig) error {
	writer := cmd.OutOrStdout()

	target, err := config.DefaultGlobalConfig()
	if err != nil {
		return err
	}
	if current != nil {
		if strings.TrimSpace(current.ScriptsRoot) != "" {
			target.ScriptsRoot = current.ScriptsRoot
		}
		if len(current.SearchPaths) > 0 {
			target.SearchPaths = current.SearchPaths
		}
	}

	result, err := bootstrap.EnsureGlobalAssets(target, true)
	if err != nil {
		return err
	}
	target.DefaultConfigPath = result.ConfigPath
	target.ScriptsRoot = result.ScriptsDir
	target.SearchPaths = bootstrap.UniqueStrings(append(target.SearchPaths, filepath.Dir(result.ConfigPath)))

	if err := bootstrap.PersistGlobalConfig(target, true); err != nil {
		return err
	}
	if err := config.SaveActiveConfigPath(result.ConfigPath); err != nil {
		return err
	}

	ui.Success(writer, "已重置全局配置路径")
	return nil
}

func buildEnvSelectionContext(cmd *cobra.Command, deps Dependencies, globalCfg *config.GlobalConfig) (*envSelectionContext, error) {
	configFlagPath, err := currentConfigPath(cmd, deps)
	if err != nil {
		return nil, err
	}
	if !pathExists(configFlagPath) {
		if legacy := projectConfigPath(deps.BaseDir); legacy != "" && pathExists(legacy) {
			configFlagPath = legacy
		}
	}

	projectDirs := projectRoots(deps.BaseDir)
	projectDir := ""
	if len(projectDirs) > 0 {
		projectDir = projectDirs[0]
	}

	homeDir, _ := config.ResolveHomeDir()
	globalConfigDir := ""
	if strings.TrimSpace(homeDir) != "" {
		globalConfigDir = filepath.Join(homeDir, "config")
	}

	searchDirs, defaultPath := collectSearchDirs(globalCfg, projectDir, globalConfigDir)

	candidates, err := listConfigFiles(searchDirs)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return &envSelectionContext{}, nil
	}

	activePath := ""
	if strings.TrimSpace(projectDir) != "" {
		if projectActive, err := config.LoadProjectActiveConfigPath(projectDir); err == nil {
			activePath = projectActive
		} else {
			return nil, err
		}
	}
	if strings.TrimSpace(activePath) == "" {
		globalActive, err := config.LoadActiveConfigPath()
		if err != nil {
			return nil, err
		}
		activePath = globalActive
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

	options, groups := buildConfigGroups(candidates, globalConfigDir, deps.BaseDir)
	return &envSelectionContext{
		options:    options,
		groups:     groups,
		activePath: activePath,
		projectDir: projectDir,
		globalDir:  globalConfigDir,
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
			meta := selectOptionTemplateMeta{
				Group:     group.Title,
				Name:      option.DisplayName,
				Path:      option.AbsolutePath,
				Active:    isActive,
				First:     isNewGroup,
				GapBefore: isNewGroup && len(options) > 0,
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
		Message:  "请选择要激活的配置文件（支持 ↑/↓ 导航，/ 搜索，Enter 确认）:",
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

func persistEnvSelection(globalCfg *config.GlobalConfig, projectDir string, globalDir string, chosen configCandidate) error {
	cleanProject := strings.TrimSpace(projectDir)
	if cleanProject != "" {
		cleanProject = filepath.Clean(cleanProject)
		if err := config.SaveProjectActiveConfigPath(cleanProject, chosen.AbsolutePath); err != nil {
			return err
		}
	}

	cleanGlobal := strings.TrimSpace(globalDir)
	if cleanGlobal != "" {
		cleanGlobal = filepath.Clean(cleanGlobal)
	}

	isProjectContext := cleanProject != ""
	isGlobalSelection := cleanGlobal != "" && insideDir(chosen.AbsolutePath, cleanGlobal)

	if !isProjectContext || isGlobalSelection {
		if err := config.SaveActiveConfigPath(chosen.AbsolutePath); err != nil {
			return err
		}
		globalCfg.DefaultConfigPath = chosen.AbsolutePath
		globalCfg.SearchPaths = bootstrap.UniqueStrings(append(globalCfg.SearchPaths, filepath.Dir(chosen.AbsolutePath)))
		if err := bootstrap.PersistGlobalConfig(globalCfg, true); err != nil {
			return fmt.Errorf("写入全局配置失败: %w", err)
		}
	}
	return nil
}

func renderSelectionResult(writer io.Writer, chosen configCandidate, loader *config.Loader, envName string) {
	fmt.Fprintln(writer, "")
	ui.Success(writer, "已切换至配置文件")
	fmt.Fprintf(writer, "  %s %s\n", ui.Gray("路径:"), ui.Cyan(chosen.AbsolutePath))
	if loader == nil {
		return
	}
	cfg, err := loader.Load(chosen.AbsolutePath, envName)
	if err != nil {
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

func buildConfigGroups(candidates []configCandidate, globalConfigDir string, baseDir string) ([]configCandidate, []configGroup) {
	options := make([]configCandidate, len(candidates))
	globalRoot := strings.TrimSpace(globalConfigDir)
	projectRoots := projectRoots(baseDir)
	grouped := map[string][]configCandidate{}
	for i, candidate := range candidates {
		candidate.Group = categorizeCandidate(candidate.AbsolutePath, globalRoot, projectRoots)
		options[i] = candidate
		grouped[candidate.Group] = append(grouped[candidate.Group], candidate)
	}
	var groups []configGroup
	orderedTitles := []string{"全局配置", "项目配置", "其他配置"}
	seen := map[string]bool{}
	for _, title := range orderedTitles {
		if entries, ok := grouped[title]; ok {
			sort.Slice(entries, func(i, j int) bool {
				return entries[i].DisplayName < entries[j].DisplayName
			})
			groups = append(groups, configGroup{Title: title, Options: entries})
			seen[title] = true
		}
	}
	for title, entries := range grouped {
		if seen[title] {
			continue
		}
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].DisplayName < entries[j].DisplayName
		})
		groups = append(groups, configGroup{Title: title, Options: entries})
	}
	return options, groups
}

func samePath(a, b string) bool {
	cleanA := filepath.Clean(a)
	cleanB := filepath.Clean(b)
	return cleanA == cleanB
}

func collectSearchDirs(cfg *config.GlobalConfig, projectDir string, globalDir string) ([]string, string) {
	cleanGlobal := filepath.Clean(strings.TrimSpace(globalDir))
	cleanProject := filepath.Clean(strings.TrimSpace(projectDir))

	var dirs []string
	if cleanGlobal != "" {
		dirs = append(dirs, cleanGlobal)
	}
	if cleanProject != "" {
		dirs = append(dirs, cleanProject)
	}
	dirs = bootstrap.UniqueStrings(dirs)

	defaultPath := ""
	if cleanProject != "" {
		defaultPath = filepath.Join(cleanProject, "demo.yaml")
	} else if cleanGlobal != "" {
		defaultPath = filepath.Join(cleanGlobal, "demo.yaml")
	}

	if cfg != nil {
		candidate := config.ExpandPath(strings.TrimSpace(cfg.DefaultConfigPath))
		if candidate != "" && dirInsideAllowed(filepath.Dir(candidate), dirs) {
			defaultPath = candidate
		}
	}
	if defaultPath == "" && cleanGlobal != "" {
		defaultPath = filepath.Join(cleanGlobal, "demo.yaml")
	}
	if defaultPath == "" && len(dirs) > 0 {
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

func pathExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	if _, err := os.Stat(path); err != nil {
		return false
	}
	return true
}

func projectConfigPath(baseDir string) string {
	dir := nearestProjectConfigDir(baseDir)
	if dir == "" {
		return ""
	}
	candidate := filepath.Join(dir, "demo.yaml")
	if pathExists(candidate) {
		return candidate
	}
	return candidate
}

func projectRoots(baseDir string) []string {
	dir := nearestProjectConfigDir(baseDir)
	if dir == "" {
		return nil
	}
	return []string{dir}
}

func nearestProjectConfigDir(baseDir string) string {
	root := strings.TrimSpace(baseDir)
	if root == "" {
		cwd, err := os.Getwd()
		if err == nil {
			root = cwd
		}
	}
	if root == "" {
		return ""
	}
	home, _ := config.ResolveHomeDir()
	homeClean := filepath.Clean(strings.TrimSpace(home))
	current := filepath.Clean(root)
	for {
		candidate := filepath.Join(current, ".alpen")
		if homeClean != "" && samePath(candidate, homeClean) {
			break
		}
		if pathExists(candidate) {
			return filepath.Clean(candidate)
		}
		if homeClean != "" && samePath(current, homeClean) {
			break
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return ""
}

func categorizeCandidate(path string, globalRoot string, projectRoots []string) string {
	clean := filepath.Clean(path)
	if insideDir(clean, globalRoot) {
		return "全局配置"
	}
	for _, root := range projectRoots {
		if insideDir(clean, root) {
			return "项目配置"
		}
	}
	return "其他配置"
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
