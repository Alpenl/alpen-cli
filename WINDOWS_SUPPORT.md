# Alpen CLI - Windows 支持

## ✨ Windows 平台特性

Alpen CLI 在 Windows 上会自动适配：
- 使用 `cmd.exe` 执行命令
- 环境变量命令自动转换为 `set` 格式
- 剪贴板功能原生支持

## 🎯 使用方法

### 在 Windows CMD 中

```cmd
C:\> alpen cc any

~ 正在执行: claudecode any
────────────────────────────────────────

✓ 已复制到剪贴板，请粘贴执行 (Ctrl+V):
set ANTHROPIC_AUTH_TOKEN=sk-20021217
set ANTHROPIC_BASE_URL=https://api.alpen-y.top/proxy/any

────────────────────────────────────────
+ 命令执行完成
  耗时: 1.2ms

C:\> REM 用户按 Ctrl+V 粘贴并执行
C:\> echo %ANTHROPIC_AUTH_TOKEN%
sk-20021217
```

### 在 PowerShell 中

PowerShell 也支持 `set` 命令（通过 CMD 兼容层），但推荐使用 PowerShell 原生语法：

```powershell
PS C:\> alpen cc any
# 输出的是 set 命令，在 PowerShell 中也能用

# 或者手动转换为 PowerShell 格式：
PS C:\> $env:ANTHROPIC_AUTH_TOKEN = "sk-20021217"
PS C:\> $env:ANTHROPIC_BASE_URL = "https://api.alpen-y.top/proxy/any"
```

## 🔧 平台差异对比

| 特性 | Linux/macOS | Windows CMD |
|------|-------------|-------------|
| Shell | `/bin/sh` | `cmd.exe` |
| 环境变量设置 | `export VAR=value` | `set VAR=value` |
| 环境变量读取 | `$VAR` | `%VAR%` |
| 剪贴板粘贴 | `Ctrl+Shift+V` | `Ctrl+V` |
| 配置目录 | `~/.alpen` | `%USERPROFILE%\.alpen` |

## 📝 配置示例

配置文件格式完全相同，Alpen 会自动处理平台差异：

`%USERPROFILE%\.alpen\config\config.yaml`:

```yaml
commands:
    claudecode:
        alias: cc
        actions:
            any:
                command: |-
                    echo "export ANTHROPIC_AUTH_TOKEN=sk-20021217"
                    echo "export ANTHROPIC_BASE_URL=https://api.alpen-y.top/proxy/any"
```

**注意**：
- 配置中仍然使用 `export` 格式（统一配置）
- Alpen 会自动转换为 Windows 的 `set` 格式

## ⚙️ 技术实现

### 自动平台检测

```go
func buildShell(command string) (string, []string) {
    if runtime.GOOS == "windows" {
        return "cmd.exe", []string{"/C", command}
    }
    return "/bin/sh", []string{"-c", command}
}
```

### 环境变量转换

```go
func convertExportCommand(exportCmd string) string {
    withoutExport := strings.TrimPrefix(exportCmd, "export ")

    if runtime.GOOS == "windows" {
        return "set " + withoutExport  // Windows 格式
    }

    return exportCmd  // Linux/macOS 格式
}
```

## 🚀 安装

### 使用安装脚本（PowerShell）

```powershell
# 下载并安装
Invoke-WebRequest -Uri "https://github.com/Alpenl/alpen-cli/releases/latest/download/alpen-windows-amd64.exe" -OutFile "$env:USERPROFILE\bin\alpen.exe"

# 添加到 PATH（如果需要）
$env:PATH += ";$env:USERPROFILE\bin"
```

### 手动安装

1. 下载最新版本的 `alpen-windows-amd64.exe`
2. 重命名为 `alpen.exe`
3. 放到 PATH 中的任意目录（如 `C:\Windows\System32` 或 `%USERPROFILE%\bin`）

## ✅ 功能清单

- ✅ 命令执行（cmd.exe）
- ✅ 环境变量自动转换（export → set）
- ✅ 剪贴板支持
- ✅ 配置文件管理
- ✅ 动态命令注册
- ✅ 插件系统
- ⚠️ PowerShell 原生语法（需手动转换）

## 🐛 已知限制

1. **PowerShell 环境变量**：
   - 输出的是 `set` 格式，在 PowerShell 中能用但不是最优
   - 推荐手动转换为 `$env:VAR = "value"` 格式

2. **路径分隔符**：
   - Go 的 `filepath` 包会自动处理，无需担心

3. **权限问题**：
   - 某些系统目录需要管理员权限
   - 建议安装到用户目录

## 📚 相关文档

- [剪贴板功能](./CLIPBOARD_ENV.md)
- [配置指南](./README.md)

---

**Windows 支持由 v0.3.0 开始提供** ✨
