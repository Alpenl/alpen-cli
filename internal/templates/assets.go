package templates

import (
	_ "embed"
	"strings"
	"text/template"
)

//go:embed config/global_demo.yaml.tmpl
var globalCommandsRaw string

//go:embed config/demo.yaml
var localCommandsRaw string

//go:embed config/global.yaml.tmpl
var globalConfigRaw string

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

//go:embed readme/global.md
var globalReadme string

// GlobalCommands 返回渲染后的全局命令配置模板
func GlobalCommands(data map[string]string) (string, error) {
	tmpl, err := template.New("global_commands").Parse(globalCommandsRaw)
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	if err := tmpl.Execute(&builder, data); err != nil {
		return "", err
	}
	return builder.String(), nil
}

// LocalCommands 返回项目级命令模板
func LocalCommands() string {
	return localCommandsRaw
}

// GlobalConfig 渲染全局配置模板
func GlobalConfig(data map[string]any) (string, error) {
	tmpl, err := template.New("global_config").Parse(globalConfigRaw)
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

// GlobalReadme 返回全局 README 模板
func GlobalReadme() string {
	return globalReadme
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
