package config

import "testing"

func TestConfigValidateSuccess(t *testing.T) {
	cfg := &Config{
		Commands: map[string]CommandSpec{
			"system": {
				Alias:       "sys",
				Description: "系统管理",
				Command:     "echo system",
				Actions: map[string]ActionSpec{
					"update": {
						Alias:       "up",
						Description: "更新系统",
						Command:     "echo update",
					},
				},
			},
			"codex": {
				Description: "与 Codex 交互",
				Command:     "echo codex",
			},
		},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected validation to pass, got error %v", err)
	}
}

func TestConfigValidateMissingCommand(t *testing.T) {
	cfg := &Config{
		Commands: map[string]CommandSpec{
			"broken": {
				Description: "无效命令",
			},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation to fail when command and actions missing")
	}
}

func TestConfigValidateAliasConflict(t *testing.T) {
	cfg := &Config{
		Commands: map[string]CommandSpec{
			"first": {
				Alias:   "dup",
				Command: "echo foo",
			},
			"second": {
				Alias:   "dup",
				Command: "echo bar",
			},
		},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected alias conflict to be detected")
	}
}

func TestCommandActionAliasConflict(t *testing.T) {
	spec := CommandSpec{
		Command: "echo ok",
		Actions: map[string]ActionSpec{
			"run": {
				Alias:   "dup",
				Command: "echo run",
			},
			"exec": {
				Alias:   "dup",
				Command: "echo exec",
			},
		},
	}
	if err := validateCommandSpec("demo", spec); err == nil {
		t.Fatalf("expected duplicate action alias to trigger validation error")
	}
}
