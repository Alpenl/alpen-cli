package executor

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/alpen/alpen-cli/internal/config"
	"github.com/alpen/alpen-cli/internal/lifecycle"
	"github.com/alpen/alpen-cli/internal/plugins"
	"github.com/alpen/alpen-cli/internal/ui"
)

// Executor 负责执行脚本命令，并在执行前后触发生命周期事件
type Executor struct {
	plugins *plugins.Registry
	logger  *log.Logger
}

// ScriptRequest 描述一次脚本执行所需的参数
type ScriptRequest struct {
	GroupName  string
	ScriptName string
	Template   config.ScriptTemplate
	ExtraArgs  []string
	ExtraEnv   map[string]string
	WorkingDir string
	DryRun     bool
}

// Result 表示脚本执行结果
type Result struct {
	ExitCode int
	Duration time.Duration
}

// NewExecutor 构造执行器
func NewExecutor(registry *plugins.Registry, logger *log.Logger) *Executor {
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
	if !req.Template.IsPlatformSupported() {
		return Result{}, fmt.Errorf("脚本 %s/%s 不支持当前平台 %s", req.GroupName, req.ScriptName, runtime.GOOS)
	}
	envMap := mergeEnv(req.Template.Env, req.ExtraEnv)
	payload := &lifecycle.Context{
		GroupName:  req.GroupName,
		ScriptName: req.ScriptName,
		Command:    req.Template.Command,
		Args:       req.ExtraArgs,
		Env:        envMap,
	}
	if err := e.plugins.Emit(ctx, lifecycle.EventBeforeExecute, payload); err != nil {
		return Result{}, err
	}
	if req.DryRun {
		fmt.Printf("%s %s\n", ui.Gray("命令:"), ui.Cyan(req.Template.Command))
		if len(req.ExtraArgs) > 0 {
			fmt.Printf("%s %s\n", ui.Gray("参数:"), ui.Cyan(strings.Join(req.ExtraArgs, " ")))
		}
		if len(envMap) > 0 {
			fmt.Println(ui.Gray("环境变量:"))
			for k, v := range req.Template.Env {
				fmt.Printf("  %s=%s\n", ui.Yellow(k), ui.Gray(v))
			}
		}
		return Result{ExitCode: 0}, nil
	}
	payload.StartAt = time.Now()
	result := Result{}

	combinedCommand := buildCommand(req.Template.Command, req.ExtraArgs)
	shell, shellArgs := buildShell(combinedCommand)

	cmd := exec.CommandContext(ctx, shell, shellArgs...)
	cmd.Env = envMapToList(envMap)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if req.WorkingDir != "" {
		cmd.Dir = req.WorkingDir
	}

	err := cmd.Run()
	payload.EndAt = time.Now()
	result.Duration = payload.EndAt.Sub(payload.StartAt)

	var exitErr *exec.ExitError
	if err != nil {
		if errors.Is(err, context.Canceled) {
			exitCode := -1
			payload.Err = err
			e.plugins.Emit(ctx, lifecycle.EventError, payload) // 最好捕获但这里忽略返回值
			return Result{ExitCode: exitCode, Duration: result.Duration}, err
		}
		if errors.As(err, &exitErr) {
			payload.Err = err
			result.ExitCode = exitErr.ExitCode()
			_ = e.plugins.Emit(ctx, lifecycle.EventError, payload)
			return result, err
		}
		payload.Err = err
		_ = e.plugins.Emit(ctx, lifecycle.EventError, payload)
		return result, err
	}
	result.ExitCode = 0
	if err := e.plugins.Emit(ctx, lifecycle.EventAfterExecute, payload); err != nil {
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
	return base + " " + joinArgs(extra)
}

func buildShell(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		return "cmd.exe", []string{"/C", command}
	}
	return "/bin/sh", []string{"-c", command}
}

func joinArgs(args []string) string {
	quoted := make([]string, 0, len(args))
	for _, arg := range args {
		quoted = append(quoted, quoteArg(arg))
	}
	return strings.Join(quoted, " ")
}

func quoteArg(arg string) string {
	if runtime.GOOS == "windows" {
		escaped := strings.ReplaceAll(arg, `"`, `\"`)
		return fmt.Sprintf(`"%s"`, escaped)
	}
	if !strings.ContainsAny(arg, " '\"\\") {
		return arg
	}
	escaped := strings.ReplaceAll(arg, `'`, `'\''`)
	return "'" + escaped + "'"
}
