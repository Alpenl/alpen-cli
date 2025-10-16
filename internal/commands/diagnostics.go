package commands

import (
	"fmt"
	"io"
	"strings"

	"github.com/alpen/alpen-cli/internal/config"
	"github.com/alpen/alpen-cli/internal/ui"
)

// renderDiagnostics 将配置合并产生的诊断信息输出到终端
func renderDiagnostics(writer io.Writer, diags []config.Diagnostic) {
	if len(diags) == 0 {
		return
	}

	fmt.Fprintln(writer, "")
	ui.Warning(writer, "检测到 %d 项配置提示: ", len(diags))
	for _, diag := range diags {
		level := strings.ToLower(strings.TrimSpace(diag.Level))
		switch level {
		case "error":
			ui.Error(writer, "  - %s", diag.Message)
		case "info":
			ui.Info(writer, "  - %s", diag.Message)
		default:
			ui.Warning(writer, "  - %s", diag.Message)
		}
	}
	fmt.Fprintln(writer, "")
}
