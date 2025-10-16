package executor

import (
	"context"
	"testing"
	"time"

	"github.com/alpen/alpen-cli/internal/plugins"
)

func TestBuildCommandWithArgs(t *testing.T) {
	cmd := buildCommand("echo", []string{"hello", "world"})
	if cmd != "echo hello world" {
		t.Fatalf("unexpected command: %s", cmd)
	}

	cmd = buildCommand("echo", []string{"hello world"})
	if cmd != "echo 'hello world'" && cmd != "echo \"hello world\"" {
		t.Fatalf("unexpected quoting: %s", cmd)
	}
}

func TestJoinArgsQuotes(t *testing.T) {
	result := joinArgs([]string{"hello", "a b", "c\"d"})
	if result == "" {
		t.Fatalf("joinArgs should not return empty string")
	}
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
