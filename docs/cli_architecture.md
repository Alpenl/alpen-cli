# Alpen CLI 方案设计

## 1. 目标与需求概述
- 构建一个易用的命令行工具，用于集中管理并快速运行内置脚本。
- 保持结构清晰、易于扩展，便于未来追加脚本类别、交互方式或插件能力。
- 支持本地开发者无脑执行常用任务，同时可以针对不同环境（开发/测试/生产）做差异化配置。

## 2. 核心设计原则
- **模块化**：命令解析、脚本注册、执行引擎彼此解耦。
- **配置驱动**：通过配置文件声明脚本，使新增脚本无需改动核心代码。
- **可扩展**：预留插件与钩子机制，允许按需增强功能（如日志上报、参数检查）。
- **可观测**：提供基础的日志与结果反馈，便于排查执行问题。
- **跨平台**：默认命令在 macOS/Linux/Windows 下可运行，避免平台特有语法。

## 3. 技术选型对比

| 方案 | 优点 | 缺点 | 适用场景 | 成本评估 |
| --- | --- | --- | --- | --- |
| Go + Cobra | 可编译为单一二进制；跨平台可靠；原生并发；依赖少 | 初期学习成本；需维护多平台构建流程 | 需要独立分发、持续扩展的企业内部 CLI | 中：需搭建 Go 构建与测试流水线 |
| Node.js + TypeScript + `commander` | 前端团队熟悉；生态丰富；通过 `node` 调用脚本灵活 | 需依赖 Node 运行时；跨平台兼容性需额外处理 | Web/前端团队常用脚本 | 中：需搭建 TS 构建链 |
| Python + Typer/Click | 语法简洁；系统普遍自带；适合 DevOps 场景 | 依赖虚拟环境；打包分发复杂 | 数据/运维团队脚本 | 低：只需解释器 |

> 结论：为提升可执行性与交付效率，本方案选择 **Go + Cobra** 作为核心技术栈，通过编译二进制覆盖 macOS/Linux/Windows，便于在团队或 CI/CD 环境中快速推广。

## 4. 推荐架构概览

```
alpen-cli/
├─ go.mod
├─ go.sum
├─ main.go                    # 程序入口，初始化根命令
├─ cmd/
│  └─ root.go                 # 根命令定义，全局参数与版本信息
├─ internal/
│  ├─ commands/
│  │  ├─ run.go               # 运行脚本命令
│  │  ├─ list.go              # 列出可用脚本
│  │  └─ init.go              # 初始化/模板生成
│  ├─ config/
│  │  ├─ loader.go            # 配置加载与环境差异处理
│  │  └─ schema.go            # 配置校验与默认值
│  ├─ executor/
│  │  └─ executor.go          # 封装脚本执行、日志、超时
│  ├─ lifecycle/
│  │  └─ hooks.go             # 生命周期钩子定义与调度
│  └─ plugins/
│     └─ registry.go          # 插件注册与管理
├─ scripts/
│  └─ scripts.yaml            # 默认脚本配置
├─ docs/
│  └─ cli_architecture.md     # 方案设计文档（当前文件）
└─ README.md
```

## 5. 模块职责说明
- **命令层（commands）**：基于 Cobra 的子命令集合，负责参数解析、校验与调用核心模块。
- **配置模块（config）**：解析 YAML/JSON 配置，支持环境覆盖、分组、标签等扩展字段，并提供 Schema 校验。
- **执行引擎（executor）**：封装 `os/exec` 调用，支持上下文取消、超时控制、实时日志、失败重试与并发执行。
- **生命周期钩子（lifecycle）**：定义 `BeforeRun`、`AfterRun`、`OnError` 等钩子，插件可订阅并增强执行过程。
- **插件机制（plugins）**：通过接口注册插件，注入配置与执行上下文，承担日志增强、结果上报、预检查等扩展职责。

## 6. 配置驱动示例

`scripts/scripts.yaml` 示例：

```yaml
groups:
  build:
    description: 构建相关命令
    scripts:
      webpack-build:
        command: yarn build
        description: 打包前端资源
        env:
          NODE_ENV: production
  ops:
    description: 运维辅助命令
    scripts:
      clean-cache:
        command: rm -rf .cache
        description: 清理缓存目录
        platforms: [darwin, linux]
```

- `command` 支持 shell 片段，默认使用 `/bin/sh` 或 Windows `cmd`。
- `env` 提供额外环境变量；`platforms` 控制可执行的平台。
- 后续可扩展字段：`tags`、`dependsOn`、`retry` 等。

## 7. 插件钩子机制
- 定义事件枚举：`EventRegistryLoaded`、`EventBeforeExecute`、`EventAfterExecute`、`EventError` 等。
- 插件实现 `Plugin` 接口（如 `type Plugin interface { Name() string; Handle(event lifecycle.Event, ctx lifecycle.Context) error }`），由核心注入日志、配置、执行上下文。
- 通过 `plugins/registry.go` 注册插件或在配置中声明启用顺序，支持按名称启停。
- 内置插件示例：
  - 日志增强插件：统一结构化输出，写入本地日志文件或 stdout。
  - 结果上报插件：在 `EventAfterExecute` 时发送 webhook 或写入监控系统。
- 预留插件配置段落，允许为插件传入自定义参数（例如告警阈值、目标地址）。

## 8. 可扩展交互能力
- **增强参数化**：支持 `alpen run <script> -- --flag value` 将参数透传给底层命令。
- **动态模板**：通过 `alpen init` 生成项目私有的 `scripts.yaml`。
- **脚本搜索与提示**：支持 fuzzy search，便于记忆。
- **组合任务**：允许定义宏命令，串行/并行执行多个脚本。

## 9. 测试与质量保障
- 单元测试：使用 Go 自带测试框架与 `testify` 等断言库覆盖 `config`、`executor`、`lifecycle` 模块。
- 集成测试：通过 `go test ./internal/commands -run TestCLI` 配合 `exec.CommandContext` 的 mock，模拟执行 YAML 配置中的脚本。
- 结构化日志：默认输出 JSON 或 key-value 日志，便于导入 ELK/Datadog 等监控平台。
- 持续集成：CI 中执行 `go test ./...`、`golangci-lint run`，并配置交叉编译产物验证不同平台兼容性。

## 10. 迭代路线建议
1. **MVP**：实现 `list`、`run` 命令，加载 YAML 配置执行脚本；完成基础日志与错误处理。
2. **增强阶段**：完善插件机制，提供内置通知/日志插件；集成配置 Schema 校验与环境覆盖。
3. **体验优化**：加入 shell 自动补全、模糊搜索、命令执行耗时统计等增强功能。
4. **分发与运维**：优化跨平台构建脚本，提供 Homebrew/apt/yum 等安装方式或内部制品仓发布。

## 11. 风险与对策
- **子进程兼容性问题**：封装 `executor` 根据 OS 选择 Shell（`/bin/sh`、`cmd.exe`、`powershell`），并在 CI 中验证不同平台行为。
- **配置膨胀**：以 Schema 校验与配置文档控制字段范围，引入默认值与 deprecation 机制，防止脚本参数失控。
- **安全风险**：限制脚本来源，只允许可信仓库提交；提供执行预览与确认机制。
- **后续演进**：保持核心模块纯粹，插件机制处理复杂定制，避免主干逻辑被侵染；通过语义化版本管理插件接口，降低升级风险。

## 12. 下一步行动清单
- 初始化 Go Modules 工程，拉起 Cobra 根命令。
- 实现脚本配置解析与 `run/list` 命令。
- 编写示例脚本配置与 README，指导团队使用。
- 建立测试与构建流水线，确保多平台编译与执行一致性。
