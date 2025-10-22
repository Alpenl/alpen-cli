package templates

import (
	_ "embed"
	"strings"
	"text/template"
)

//go:embed config/commands_demo.yaml.tmpl
var defaultCommandsRaw string

//go:embed config/demo_module.yaml.tmpl
var demoModuleConfigRaw string

//go:embed scripts/demo.sh
var demoShellScript string

//go:embed scripts/demo.py
var demoPythonScript string

//go:embed scripts/tests/demo.sh
var demoTestScript string

//go:embed scripts/demo_module.sh
var demoModuleScript string

//go:embed readme/home.md
var homeReadme string

// DefaultCommands 返回渲染后的命令示例模板
func DefaultCommands(data map[string]string) (string, error) {
	tmpl, err := template.New("default_commands").Parse(defaultCommandsRaw)
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	if err := tmpl.Execute(&builder, data); err != nil {
		return "", err
	}
	return builder.String(), nil
}

// DemoShell 返回示例 Bash 脚本
func DemoShell() string {
	return demoShellScript
}

// DemoPython 返回示例 Python 脚本
func DemoPython() string {
	return demoPythonScript
}

// DemoTest 返回示例测试脚本
func DemoTest() string {
	return demoTestScript
}

// HomeReadme 返回用户目录 README 模板
func HomeReadme() string {
	return homeReadme
}

// DemoModuleConfig 返回演示模块命令配置
func DemoModuleConfig(data map[string]string) (string, error) {
	tmpl, err := template.New("demo_module").Parse(demoModuleConfigRaw)
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	if err := tmpl.Execute(&builder, data); err != nil {
		return "", err
	}
	return builder.String(), nil
}

// DemoModuleScript 返回演示模块脚本
func DemoModuleScript() string {
	return demoModuleScript
}
