# Alpen CLI

Alpen CLI 是一个"配置即命令树"的统一脚本入口，使用 Go + Cobra 构建，通过 YAML 描述 CLI 结构即可生成完整的命令体验。与传统的"脚本列表 + 菜单"模式相比，重构后的版本直接将配置映射为命令层级，极大提升了可读性、可扩展性与上手速度。

## 安装

### 一键安装（推荐）

```bash
curl -fsSL https://raw.githubusercontent.com/alpenl/alpen-cli/main/install.sh | sudo bash
```

### 手动安装

1. 前往 [Releases 页面](https://github.com/alpenl/alpen-cli/releases/latest) 下载最新的 `.deb` 包
2. 安装：
   ```bash
   sudo dpkg -i alpen-cli_*.deb
   sudo apt-get install -f  # 如有依赖问题
   ```

### 从源码构建

```bash
git clone https://github.com/alpenl/alpen-cli.git
cd alpen-cli
go build -o alpen .
sudo mv alpen /usr/bin/
```

## 快速开始

```bash
# 初始化命令配置（默认写入 ~/.alpen/config/demo.yaml，生成示例脚本）
alpen init

# 查看动态生成的命令
alpen help

# 使用交互式菜单快速体验
alpen interactive

# 直接运行常用命令
alpen ls
alpen <your-command>
alpen <your-command> <action>
```

> 提示：环境变量 `ALPEN_HOME` 会在 CLI 启动时自动注入，指向 `~/.alpen` 目录，可在脚本与配置中复用。
> 限制：CLI 仅识别 `~/.alpen` 目录内的配置文件，如需切换请将文件放置于该目录并通过 `--config` 选项指定。

## 命令模型概览

- 顶层命令对应 YAML 中的 `commands` 键，例如 `deploy`、`build`。
- 若命令提供 `command` 字段，则 `alpen <name>` 会执行该脚本。
- `actions` 下的子项会转成二级命令，例如 `alpen deploy release`。
- 命令和子命令都支持 `alias`，可设置更短的调用方式。
- 所有额外参数（包括 `--` 之后的内容）都会原样透传到底层脚本。
- 使用 `alpen ls` 或 `alpen <命令> ls` 可查看配置中的命令简介，便于快速检索。

## 示例配置（默认位于 `~/.alpen/config/demo.yaml`）

```yaml
commands:
  demo:
    description: 运行示例脚本
    command: "$ALPEN_HOME/config/scripts/demo.sh"
    actions:
      smoke-test:
        description: 运行示例测试脚本
        command: "$ALPEN_HOME/config/scripts/tests/demo.sh"
```

- `commands` 中的键就是一级命令，按需增删即可。
- `alias` 可选，用于提供形如 `alpen sys update` 的缩写。
- 若某命令只需子命令，可省略顶层 `command` 字段。
- 支持通过 `demo.<env>.yaml`（与主配置同目录）覆盖差异配置，运行时使用 `--environment` 指定环境。

## 常用操作

- `alpen init`：在 `~/.alpen/config/demo.yaml` 写入示例配置并生成脚本模板（支持 `--force` 覆盖）。
- `alpen help`：查看当前命令树。
- `alpen env` / `alpen -e`：在 `~/.alpen/config` 下选择并激活配置文件。
- `alpen ls`：列出配置中的顶层命令；`alpen <cmd> ls` 查看子命令。
- `alpen interactive`：以交互式菜单方式选择命令，支持输入额外参数。
- `alpen --config ~/.alpen/config/xxx.yaml`：在 `~/.alpen/config` 下切换其他配置文件，可与 `--environment` 搭配使用。
- `alpen <cmd> [args]`：执行某个命令，`args` 会透传到底层脚本。
- `alpen <cmd> <action> -- --flag`：执行子命令并透传参数，例如 `alpen deploy release -- --help`。
- `alpen script ls`：查看脚本仓库文件。
- `alpen script doctor`：检查脚本可执行权限与 Shebang。
- `alpen version` / `alpen -v`：查看构建版本信息。
- `go test ./...`：运行自动化测试。
- `golangci-lint run`：静态分析与格式检查（已在 CI 中集成）。

## 开发指南

### Git Hooks 配置

项目提供了自动化的代码质量检查 Git Hooks，每次提交时会自动运行：

1. **安装 Git Hooks**（首次克隆仓库后执行）：
   ```bash
   ./scripts/install-hooks.sh
   ```

2. **Pre-commit Hook 会自动检查**：
   - ✅ `go fmt` - 代码格式化检查
   - ✅ `go vet` - 静态分析
   - ✅ `golangci-lint` - 代码质量检查（如已安装）

3. **如需跳过检查**（紧急情况）：
   ```bash
   git commit --no-verify
   ```

4. **推荐安装 golangci-lint**：
   ```bash
   curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
   ```

5. **卸载 Hooks**：
   ```bash
   rm .git/hooks/pre-commit
   ```

> **提示**：Hooks 脚本位于 `scripts/git-hooks/` 目录，由版本控制管理，团队成员可统一更新。

### 本地打包测试

验证 DEB 包构建和内容，确保不包含无关文件：

```bash
# 构建并验证 DEB 包
./scripts/build-and-verify.sh

# 查看包内容
dpkg-deb --contents alpen-cli_*.deb

# 本地测试安装
sudo dpkg -i alpen-cli_*.deb
alpen --version
```

打包规则：
- ✅ 仅打包 `alpen` 二进制和必要文档
- ❌ 排除 `node_modules/`、`.git/`、`scripts/` 等开发文件
- 📋 完整排除列表参见 `.debignore`

## 目录结构

```
.
├─ cmd/              # CLI 入口与根命令初始化
├─ internal/
│  ├─ commands/      # 动态命令注册、init 等内置子命令
│  ├─ config/        # demo.yaml 解析与合并
│  ├─ executor/      # 命令执行器与生命周期事件
│  ├─ lifecycle/     # 生命周期事件模型
│  └─ plugins/       # 插件注册与调度
├─ dist/             # 编译产物
└─ docs/             # 设计与重构文档
```

## 后续计划

- 丰富命令描述字段（环境变量、工作目录、平台约束等）并提供 Schema 校验。
- 提供插件示例，支持执行日志、结果上报等扩展。
- 补充测试与 CI 流程，确保动态命令在多平台行为一致。
