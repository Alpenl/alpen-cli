# Alpen CLI

Alpen CLI 是一个通过配置驱动的脚本统一执行工具，帮助团队在本地或 CI 环境里快速调用常用脚本。项目使用 Go + Cobra 构建，可编译为跨平台二进制，便于分发与部署，并提供交互式菜单以便快速选择脚本。

## 快速开始

```bash
# 拉取依赖并一次性编译
go mod tidy
go build -o dist/alpen ./...

# 初始化示例配置
./dist/alpen init

# 通过菜单启动（默认）
./dist/alpen

# 查看可用脚本列表
./dist/alpen list

# 执行脚本
./dist/alpen run linux-tools/codex-run -- --help
```

> 提示：`go run .` 每次都会重新编译，适合临时验证；日常使用建议直接运行 `dist/alpen` 以获得更快的启动速度。

## 将二进制加入 PATH

- 开发环境建议优先使用 `dist/alpen`，也可以将其复制或软链到 `/usr/local/bin/alpen` 等目录，方便直接输入 `alpen`。
- 若需交叉编译，可使用 `GOOS`/`GOARCH` 变量，例如 `GOOS=darwin GOARCH=arm64 go build -o dist/alpen-darwin ./...` 并分发对应产物。

## 交互式菜单

- 直接运行 `alpen` 会展示顶层菜单，例如：
  - `1. alpen a - 常用的初始化与环境准备动作`
  - `2. alpen b - 常见工具与脚本快捷入口`
- 在菜单界面输入序号或菜单 key（例如 `1`、`a`、`alpen a`）并回车即可进入；输入 `q` 返回或退出。
- 输入 `alpen a` 会进入对应子菜单，列出 `linux-init` 分组下的所有脚本，选择后即可执行。
- 输入 `alpen b` 会展示自定义快捷项，例如：
  - `1. 安装 Codex CLI`
  - `2. 运行 Codex CLI`
- 子菜单可配置别名：如 `alpen b codex -i` 将直接执行 “安装 Codex”，`alpen b codex` 将执行 “运行 Codex”。多余的参数会在匹配别名后透传给脚本。

## 核心命令

- `alpen list`：展示 `scripts/scripts.yaml` 中的所有脚本，支持 `--group` 过滤。
- `alpen run <脚本名>`：执行脚本，可通过 `组/脚本名` 精确指定，同时支持：
  - `--group` 指定分组；
  - `--dry-run` 仅输出计划，不实际执行；
  - `--env KEY=VALUE` 注入额外环境变量；
  - `--dir` 自定义工作目录；
  - 使用 `--` 将参数透传给底层命令，例如 `alpen run linux-tools/codex-run -- --help`。
- `alpen init`：生成默认的 `scripts/scripts.yaml`，可以使用 `--force` 覆盖已有文件。
- `alpen version`：查看版本信息。

## 配置文件

默认配置文件位置为 `scripts/scripts.yaml`，结构示例：

```yaml
groups:
  linux-init:
    description: Linux 初始化命令
    scripts:
      update-system:
        command: sudo apt update && sudo apt upgrade -y
        description: 更新系统软件包
  linux-tools:
    description: 常用 Linux 工具脚本
    scripts:
      codex-run:
        command: codex --dangerously-bypass-approvals-and-sandbox
        description: 运行 Codex CLI（启动时跳过审批）
menus:
  - key: "a"
    title: Linux 初始化命令
    description: 常用的初始化与环境准备动作
    group: linux-init
  - key: "b"
    title: 常用 Linux 命令
    description: 常见工具与脚本快捷入口
    items:
      - key: "codex -i"
        label: 安装 Codex CLI
        script: linux-tools/codex-install
        aliases:
          - codex install
      - key: "codex"
        label: 运行 Codex CLI
        script: linux-tools/codex-run
```

- `command`：实际执行的 shell 语句。
- `env`：执行时注入的环境变量。
- `platforms`：可选字段，限制脚本运行的平台（`darwin`/`linux`/`windows`）。
- `menus`：定义顶层与子菜单，`key` 对应命令别名，例如 `alpen b`；`group` 表示自动列出某分组内的脚本；`items` 可单独列出脚本并配置 `aliases`。

当需要根据环境加载差异化配置时，可在 `scripts/scripts.<env>.yaml` 中声明覆盖字段，并运行时通过 `-e`/`--environment` 指定环境。

## 目录结构

```
.
├─ cmd/              # CLI 命令入口
├─ internal/
│  ├─ commands/      # Cobra 子命令具体实现
│  ├─ config/        # 配置解析与合并逻辑
│  ├─ executor/      # 脚本执行器与生命周期调度
│  ├─ lifecycle/     # 生命周期事件定义
│  └─ plugins/       # 插件注册与调度
├─ scripts/          # 默认脚本配置
├─ dist/             # 已编译二进制（默认忽略，可自行发布）
└─ docs/             # 设计文档
```

## 后续计划

- 完成插件样例（日志增强、执行结果上报等）。
- 引入配置 Schema 校验与更丰富的字段（依赖、重试策略）。
- 增加测试覆盖率和 CI 脚本，保证跨平台行为一致。
