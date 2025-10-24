package executor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"
	"github.com/kballard/go-shellquote"

	"github.com/alpen/alpen-cli/internal/lifecycle"
	"github.com/alpen/alpen-cli/internal/plugins"
	"github.com/alpen/alpen-cli/internal/scripts"
	"github.com/alpen/alpen-cli/internal/ui"
)

// Executor 负责执行脚本命令，并在执行前后触发生命周期事件
type Executor struct {
	plugins  *plugins.Registry
	logger   *log.Logger
	rootOnce sync.Once
	rootPath string
	rootErr  error
}

// ScriptRequest 描述一次命令执行所需的参数
type ScriptRequest struct {
	CommandPath []string
	Command     string
	BaseEnv     map[string]string
	ExtraArgs   []string
	ExtraEnv    map[string]string
	WorkingDir  string
	DryRun      bool
}

// Result 表示脚本执行结果
type Result struct {
	ExitCode int
	Duration time.Duration
}

// NewExecutor 构造执行器
func NewExecutor(registry *plugins.Registry, logger *log.Logger) *Executor {
	if registry == nil {
		registry = plugins.NewRegistry()
	}
	if logger == nil {
		logger = log.New(os.Stdout, "[alpen] ", log.LstdFlags)
	}
	return &Executor{
		plugins: registry,
		logger:  logger,
	}
}

// Execute 运行脚本并在过程中派发事件
func (e *Executor) Execute(ctx context.Context, req ScriptRequest) (Result, error) {
	pathLabel := strings.Join(req.CommandPath, " ")
	if strings.TrimSpace(pathLabel) == "" {
		pathLabel = "<anonymous>"
	}
	if strings.TrimSpace(req.Command) == "" {
		err := errors.New("command 不能为空")
		e.logger.Printf("执行失败 path=%s err=%v", pathLabel, err)
		return Result{}, err
	}
	if err := e.validateScriptCommand(req); err != nil {
		e.logger.Printf("脚本校验失败 path=%s err=%v", pathLabel, err)
		return Result{}, err
	}
	envMap := mergeEnv(req.BaseEnv, req.ExtraEnv)
	payload := &lifecycle.Context{
		CommandPath: req.CommandPath,
		Command:     req.Command,
		Args:        req.ExtraArgs,
		Env:         envMap,
	}
	setLegacyNames(payload, req.CommandPath)
	if err := e.plugins.Emit(ctx, lifecycle.EventBeforeExecute, payload); err != nil {
		e.logger.Printf("执行前置钩子失败 path=%s err=%v", pathLabel, err)
		return Result{}, err
	}
	if req.DryRun {
		e.logger.Printf("DryRun path=%s command=%s args=%v", pathLabel, req.Command, req.ExtraArgs)
		ui.KeyValue(os.Stdout, "命令", req.Command)
		if len(req.ExtraArgs) > 0 {
			ui.KeyValue(os.Stdout, "参数", strings.Join(req.ExtraArgs, " "))
		}
		if len(req.BaseEnv) > 0 || len(req.ExtraEnv) > 0 {
			fmt.Fprintln(os.Stdout, ui.Gray("  环境变量:"))
			for k, v := range req.BaseEnv {
				fmt.Fprintf(os.Stdout, "    %s=%s\n", ui.Yellow(k), ui.Gray(v))
			}
			for k, v := range req.ExtraEnv {
				fmt.Fprintf(os.Stdout, "    %s=%s\n", ui.Yellow(k), ui.Gray(v))
			}
		}
		return Result{ExitCode: 0}, nil
	}
	payload.StartAt = time.Now()
	result := Result{}

	combinedCommand := buildCommand(req.Command, req.ExtraArgs)
	shell, shellArgs := buildShell(combinedCommand)

	cmd := exec.CommandContext(ctx, shell, shellArgs...)
	cmd.Env = envMapToList(envMap)

	// 捕获命令输出以便复制到剪贴板
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	cmd.Stdin = os.Stdin
	if req.WorkingDir != "" {
		cmd.Dir = req.WorkingDir
	}

	err := cmd.Run()
	payload.EndAt = time.Now()
	result.Duration = payload.EndAt.Sub(payload.StartAt)

	// 处理输出
	stdoutStr := stdoutBuf.String()
	stderrStr := stderrBuf.String()

	// 如果是环境变量命令，只显示剪贴板提示，不重复输出
	if err == nil && isEnvCommand(req.CommandPath) {
		clipboardContent := extractExportCommands(stdoutStr)
		if clipboardContent != "" {
			// 复制到剪贴板
			if clipErr := clipboard.WriteAll(clipboardContent); clipErr == nil {
				// 简洁的提示
				fmt.Fprintln(os.Stdout, "")
				fmt.Fprintln(os.Stdout, ui.Green("✓ 已复制到剪贴板，请粘贴执行 (Ctrl+Shift+V):"))
				fmt.Fprintln(os.Stdout, ui.Gray(clipboardContent))
			} else {
				// 复制失败，显示命令让用户手动复制
				fmt.Fprintln(os.Stdout, "")
				fmt.Fprintln(os.Stdout, ui.Yellow("请手动复制以下命令："))
				fmt.Fprintln(os.Stdout, clipboardContent)
			}
		}
	} else {
		// 其他命令正常输出
		if stdoutStr != "" {
			fmt.Print(stdoutStr)
		}
	}

	// 错误输出始终显示
	if stderrStr != "" {
		fmt.Fprint(os.Stderr, stderrStr)
	}

	var exitErr *exec.ExitError
	if err != nil {
		if errors.Is(err, context.Canceled) {
			exitCode := -1
			payload.Err = err
			_ = e.plugins.Emit(ctx, lifecycle.EventError, payload) // 忽略错误,因为主流程已被取消
			e.logger.Printf("命令被取消 path=%s err=%v", pathLabel, err)
			return Result{ExitCode: exitCode, Duration: result.Duration}, err
		}
		if errors.As(err, &exitErr) {
			payload.Err = err
			result.ExitCode = exitErr.ExitCode()
			_ = e.plugins.Emit(ctx, lifecycle.EventError, payload)
			e.logger.Printf("命令执行失败 path=%s exit=%d err=%v", pathLabel, result.ExitCode, err)
			return result, err
		}
		payload.Err = err
		_ = e.plugins.Emit(ctx, lifecycle.EventError, payload)
		e.logger.Printf("命令执行失败 path=%s err=%v", pathLabel, err)
		return result, err
	}
	result.ExitCode = 0
	if err := e.plugins.Emit(ctx, lifecycle.EventAfterExecute, payload); err != nil {
		e.logger.Printf("执行后置钩子失败 path=%s err=%v", pathLabel, err)
		return result, err
	}
	return result, nil
}

func mergeEnv(base map[string]string, override map[string]string) map[string]string {
	envMap := map[string]string{}
	for _, pair := range os.Environ() {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	for k, v := range base {
		envMap[k] = v
	}
	for k, v := range override {
		envMap[k] = v
	}
	return envMap
}

func setLegacyNames(payload *lifecycle.Context, path []string) {
	if len(path) == 0 {
		payload.GroupName = ""
		payload.ScriptName = ""
		return
	}
	payload.GroupName = path[0]
	payload.ScriptName = path[len(path)-1]
}

func envMapToList(envMap map[string]string) []string {
	result := make([]string, 0, len(envMap))
	for k, v := range envMap {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

func buildCommand(base string, extra []string) string {
	if len(extra) == 0 {
		return base
	}
	// 使用 shellquote.Join 正确转义参数,防止命令注入
	return base + " " + shellquote.Join(extra...)
}

func buildShell(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd.exe", []string{"/C", command}
	}
	return "/bin/sh", []string{"-c", command}
}

func (e *Executor) validateScriptCommand(req ScriptRequest) error {
	tokens, err := shellquote.Split(req.Command)
	if err != nil {
		return fmt.Errorf("解析命令 %q 失败: %w", req.Command, err)
	}
	if len(tokens) == 0 {
		return fmt.Errorf("命令 %q 解析后为空", req.Command)
	}
	token := os.ExpandEnv(tokens[0])
	if token == "" {
		return nil
	}
	scriptPath, relevant, err := scripts.ResolveCommandTarget(token, req.WorkingDir)
	if err != nil || !relevant {
		return err
	}
	root, err := e.resolveScriptsRoot()
	if err != nil {
		return err
	}
	scriptPath = filepath.Clean(scriptPath)
	if !scripts.IsUnderRoot(scriptPath, root) {
		return nil
	}
	return scripts.VerifyExecutable(scriptPath)
}

func (e *Executor) resolveScriptsRoot() (string, error) {
	e.rootOnce.Do(func() {
		e.rootPath, e.rootErr = scripts.ResolveRoot()
	})
	return e.rootPath, e.rootErr
}

// isEnvCommand 判断是否是环境变量设置命令
func isEnvCommand(commandPath []string) bool {
	if len(commandPath) < 2 {
		return false
	}
	// 检查是否是 cc any/yaya 或 claudecode any/yaya
	firstCmd := commandPath[0]
	secondCmd := commandPath[1]

	if (firstCmd == "cc" || firstCmd == "claudecode") && (secondCmd == "any" || secondCmd == "yaya") {
		return true
	}
	return false
}

// extractExportCommands 从输出中提取 export 命令，并根据平台转换格式
func extractExportCommands(output string) string {
	var exports []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "export ") {
			// 转换为当前平台的格式
			converted := convertExportCommand(trimmed)
			exports = append(exports, converted)
		}
	}
	if len(exports) == 0 {
		return ""
	}
	return strings.Join(exports, "\n")
}

// convertExportCommand 将 export 命令转换为当前平台格式
func convertExportCommand(exportCmd string) string {
	// 移除 "export " 前缀
	withoutExport := strings.TrimPrefix(exportCmd, "export ")

	if runtime.GOOS == "windows" {
		// Windows CMD 格式: set VAR=value
		return "set " + withoutExport
	}

	// Linux/macOS 保持原样
	return exportCmd
}
