package executor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alpen/alpen-cli/internal/lifecycle"
	"github.com/alpen/alpen-cli/internal/plugins"
)

func TestBuildCommandWithArgs(t *testing.T) {
	cmd := buildCommand("echo", []string{"hello", "world"})
	if cmd != "echo hello world" {
		t.Fatalf("unexpected command: %s", cmd)
	}

	cmd = buildCommand("echo", []string{"hello world"})
	if cmd != "echo 'hello world'" {
		t.Fatalf("unexpected quoting: %s", cmd)
	}
}

func TestExecutorCommandInjectionPrevention(t *testing.T) {
	type testCase struct {
		name    string
		pattern string
	}
	cases := []testCase{
		{name: "semicolon", pattern: "; touch %s"},
		{name: "pipe", pattern: "| touch %s"},
		{name: "and-operator", pattern: "&& touch %s"},
		{name: "dollar-substitution", pattern: "$(touch %s)"},
		{name: "backtick-substitution", pattern: "`touch %s`"},
	}

	tempDir := t.TempDir()
	exec := NewExecutor(plugins.NewRegistry(), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sentinel := filepath.Join(tempDir, fmt.Sprintf("compromised %s marker", tc.name))
			// 确保测试前的哨兵文件不存在
			if err := os.Remove(sentinel); err != nil && !os.IsNotExist(err) {
				t.Fatalf("清理哨兵文件失败: %v", err)
			}

			result, err := exec.Execute(ctx, ScriptRequest{
				CommandPath: []string{"test", "injection"},
				Command:     "echo SAFE",
				ExtraArgs:   []string{fmt.Sprintf(tc.pattern, sentinel)},
			})
			if err != nil {
				t.Fatalf("执行命令失败: %v", err)
			}
			if result.ExitCode != 0 {
				t.Fatalf("命令退出码异常: %d", result.ExitCode)
			}

			// 若注入被执行哨兵文件会被创建,这里确保文件依旧不存在
			if _, err := os.Stat(sentinel); err == nil {
				t.Fatalf("检测到命令注入,哨兵文件被创建: %s", sentinel)
			} else if !os.IsNotExist(err) {
				t.Fatalf("检查哨兵文件时出错: %v", err)
			}
		})
	}
}

func TestExecutorEnvironmentVariables(t *testing.T) {
	exec := NewExecutor(plugins.NewRegistry(), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 测试环境变量合并优先级: ExtraEnv > BaseEnv > System Env
	result, err := exec.Execute(ctx, ScriptRequest{
		CommandPath: []string{"test", "env"},
		Command:     "echo test",
		BaseEnv: map[string]string{
			"TEST_VAR":  "base_value",
			"ONLY_BASE": "base_only",
		},
		ExtraEnv: map[string]string{
			"TEST_VAR":   "extra_value", // 应该覆盖 BaseEnv
			"ONLY_EXTRA": "extra_only",
		},
		DryRun: true, // 使用 DryRun 避免实际执行
	})

	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("unexpected exit code: %d", result.ExitCode)
	}

	// 测试 mergeEnv 函数
	merged := mergeEnv(
		map[string]string{"KEY1": "base", "KEY2": "base"},
		map[string]string{"KEY2": "override", "KEY3": "extra"},
	)

	if merged["KEY1"] != "base" {
		t.Errorf("expected KEY1=base, got %s", merged["KEY1"])
	}
	if merged["KEY2"] != "override" {
		t.Errorf("expected KEY2=override, got %s", merged["KEY2"])
	}
	if merged["KEY3"] != "extra" {
		t.Errorf("expected KEY3=extra, got %s", merged["KEY3"])
	}
}

func TestExecutorContextCancellation(t *testing.T) {
	exec := NewExecutor(plugins.NewRegistry(), nil)
	ctx, cancel := context.WithCancel(context.Background())

	// 立即取消 context
	cancel()

	// 执行一个会运行较长时间的命令
	_, err := exec.Execute(ctx, ScriptRequest{
		CommandPath: []string{"test", "cancel"},
		Command:     "sleep 10",
	})

	// 应该返回 context.Canceled 错误
	if err == nil {
		t.Fatalf("expected context cancellation error, got nil")
	}
	if !strings.Contains(err.Error(), "context canceled") && !strings.Contains(err.Error(), "signal: killed") {
		t.Logf("got error: %v (acceptable if process was killed)", err)
	}
}

func TestExecutorLifecycleHookFailure(t *testing.T) {
	registry := plugins.NewRegistry()

	// 创建一个会失败的测试插件
	failPlugin := &testPlugin{
		name: "test-fail-plugin",
		handler: func(ctx context.Context, event lifecycle.Event, payload *lifecycle.Context) error {
			if event == lifecycle.EventBeforeExecute {
				return fmt.Errorf("模拟钩子失败")
			}
			return nil
		},
	}

	if err := registry.Register(failPlugin); err != nil {
		t.Fatalf("注册插件失败: %v", err)
	}

	exec := NewExecutor(registry, nil)
	ctx := context.Background()

	_, err := exec.Execute(ctx, ScriptRequest{
		CommandPath: []string{"test", "hook"},
		Command:     "echo test",
	})

	// 应该因为前置钩子失败而返回错误
	if err == nil {
		t.Fatalf("expected hook failure error, got nil")
	}
	if !strings.Contains(err.Error(), "模拟钩子失败") {
		t.Errorf("expected error to contain '模拟钩子失败', got: %v", err)
	}
}

// 测试插件辅助结构
type testPlugin struct {
	name    string
	handler func(ctx context.Context, event lifecycle.Event, payload *lifecycle.Context) error
}

func (p *testPlugin) Name() string {
	return p.name
}

func (p *testPlugin) Handle(ctx context.Context, event lifecycle.Event, payload *lifecycle.Context) error {
	if p.handler != nil {
		return p.handler(ctx, event, payload)
	}
	return nil
}

func TestExecutorExecuteEmptyCommand(t *testing.T) {
	exec := NewExecutor(plugins.NewRegistry(), nil)
	_, err := exec.Execute(context.Background(), ScriptRequest{})
	if err == nil {
		t.Fatalf("expected error for empty command")
	}
	if err.Error() != "command 不能为空" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestExecutorDryRun(t *testing.T) {
	exec := NewExecutor(plugins.NewRegistry(), nil)
	result, err := exec.Execute(context.Background(), ScriptRequest{
		CommandPath: []string{"demo"},
		Command:     "echo ok",
		DryRun:      true,
		ExtraArgs:   []string{"--flag"},
	})
	if err != nil {
		t.Fatalf("dry run should not return error, got %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("unexpected exit code: %d", result.ExitCode)
	}
}

func TestExecutorSuccessfullyRunsCommand(t *testing.T) {
	exec := NewExecutor(plugins.NewRegistry(), nil)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := exec.Execute(ctx, ScriptRequest{
		CommandPath: []string{"echo"},
		Command:     "echo success",
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("unexpected exit code: %d", result.ExitCode)
	}
	if result.Duration <= 0 {
		t.Fatalf("expected positive duration")
	}
}
