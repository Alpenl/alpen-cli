# Alpen CLI

Alpen CLI 是一个"配置即命令树"的统一脚本入口，使用 Go + Cobra 构建，通过 YAML 描述 CLI 结构即可生成完整的命令体验。与传统的"脚本列表 + 菜单"模式相比，重构后的版本直接将配置映射为命令层级，极大提升了可读性、可扩展性与上手速度。

---

## 📦 安装

### 一键安装（推荐）

**国内用户**：
```bash
# 克隆并安装（自动使用镜像加速）
git clone --branch main --depth 1 \
  https://gh-proxy.com/https://github.com/Alpenl/alpen-cli.git && \
  cd alpen-cli && \
  sudo CHINA_MIRROR=1 bash install.sh
```

**国外用户**：
```bash
# 在线安装
curl -fsSL https://raw.githubusercontent.com/alpenl/alpen-cli/main/install.sh | \
  sudo bash
```

> 💡 **提示**：安装脚本会自动检测网络环境。国内用户建议使用 `CHINA_MIRROR=1` 强制镜像模式以确保下载速度。

---

### 手动安装

1. 前往 [Releases 页面](https://github.com/alpenl/alpen-cli/releases/latest) 下载最新的 `.deb` 包
2. 安装：
   ```bash
   sudo dpkg -i alpen-cli_*.deb
   sudo apt-get install -f  # 如有依赖问题
   ```

---

### 从源码构建

```bash
git clone https://github.com/alpenl/alpen-cli.git
cd alpen-cli

# 方式1：使用构建脚本（推荐，自动注入版本信息）
./scripts/build.sh

# 方式2：手动构建
go build -o alpen .
sudo mv alpen /usr/bin/
```

> 💡 **版本管理说明**：
> - 版本号由 Git tag 统一管理，不在代码中硬编码
> - `./scripts/build.sh` 会自动从 Git 获取版本信息并注入到二进制
> - CI/CD 构建时自动使用 tag 版本号

---

### 安装选项

**环境变量**：

- **`CHINA_MIRROR=1`** - 强制使用国内镜像（**推荐国内用户**）
  ```bash
  # 本地安装
  sudo CHINA_MIRROR=1 bash install.sh

  # 在线安装
  curl -fsSL https://raw.githubusercontent.com/alpenl/alpen-cli/main/install.sh | \
    sudo CHINA_MIRROR=1 bash
  ```

- **调试模式** - 查看详细的下载和安装过程
  ```bash
  sudo bash -x install.sh
  ```

> 💡 **国内用户建议**：如果自动检测不准确或下载速度慢，请使用 `CHINA_MIRROR=1` 强制镜像模式

---

## 🚀 快速开始

```bash
# 初始化命令配置（生成示例）
alpen init

# 查看动态生成的命令
alpen help

# 使用交互式菜单（推荐新手）
alpen ui

# 直接运行命令
alpen ls
alpen <your-command>
alpen <your-command> <action>
```

> 💡 **提示**：
> - 环境变量 `ALPEN_HOME` 会自动注入，指向 `~/.alpen` 目录
> - CLI 仅识别 `~/.alpen` 目录内的配置文件

---

## 📖 命令模型

Alpen CLI 采用"配置即命令"的设计理念：

- **顶层命令**：对应 YAML 中的 `commands` 键（如 `deploy`、`build`）
- **默认动作**：若命令提供 `command` 字段，则 `alpen <name>` 直接执行该脚本
- **子命令**：`actions` 下的子项转为二级命令（如 `alpen deploy release`）
- **别名支持**：命令和子命令都支持 `alias`，可设置更短的调用方式
- **参数透传**：所有额外参数会原样透传到底层脚本
- **命令列表**：使用 `alpen ls` 或 `alpen <命令> ls` 快速查看

---

## ⚙️ 示例配置

默认位于 `~/.alpen/config/demo.yaml`：

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

**配置说明**：
- `commands` 中的键就是一级命令，按需增删
- `alias` 可选，用于提供缩写（如 `alpen sys update`）
- 若某命令只需子命令，可省略顶层 `command` 字段
- 支持环境差异配置：`demo.<env>.yaml`，通过 `--environment` 指定

---

## 🔧 常用操作

### 基础命令

| 命令 | 说明 |
|------|------|
| `alpen init` | 初始化示例配置（支持 `--force` 覆盖） |
| `alpen help` | 查看当前命令树 |
| `alpen env` / `alpen -e` | 选择并激活配置文件 |
| `alpen ls` | 列出顶层命令 |
| `alpen <cmd> ls` | 查看子命令 |
| `alpen ui` | 交互式菜单导航 |
| `alpen version` / `alpen -v` | 查看版本信息 |

### 高级用法

```bash
# 切换配置文件
alpen --config ~/.alpen/config/xxx.yaml

# 指定环境
alpen --environment prod

# 执行命令并透传参数
alpen <cmd> [args]
alpen <cmd> <action> -- --flag

# 脚本仓库管理
alpen script ls      # 查看脚本文件
alpen script doctor  # 检查脚本权限
```

---

## 🛠️ 开发指南

### Git Hooks 配置

项目提供自动化代码质量检查，每次提交时运行：

**1. 安装 Git Hooks**（首次克隆后执行）：
```bash
./scripts/install-hooks.sh
```

**2. Pre-commit Hook 自动检查**：
- ✅ `go fmt` - 代码格式化
- ✅ `go vet` - 静态分析
- ✅ `golangci-lint` - 代码质量检查（如已安装）

**3. 跳过检查**（紧急情况）：
```bash
git commit --no-verify
```

**4. 安装 golangci-lint**（推荐）：
```bash
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
  sh -s -- -b $(go env GOPATH)/bin
```

**5. 卸载 Hooks**：
```bash
rm .git/hooks/pre-commit
```

> **提示**：Hooks 脚本位于 `scripts/git-hooks/` 目录，由版本控制管理，团队成员可统一更新。

---

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

**打包规则**：
- ✅ 仅打包 `alpen` 二进制和必要文档
- ❌ 排除 `node_modules/`、`.git/`、`scripts/` 等开发文件
- 📋 完整排除列表参见 `.debignore`

---

### 测试与代码检查

```bash
# 运行单元测试
go test -v ./...

# 静态代码检查
golangci-lint run

# 本地构建（自动注入版本）
./scripts/build.sh
```

---

## 📁 目录结构

```
.
├── cmd/                    # CLI 入口与根命令初始化
├── internal/
│   ├── commands/           # 动态命令注册、内置子命令
│   ├── config/             # YAML 解析与配置合并
│   ├── executor/           # 命令执行器与生命周期
│   ├── lifecycle/          # 生命周期事件模型
│   ├── plugins/            # 插件注册与调度
│   ├── scripts/            # 脚本管理
│   ├── templates/          # 配置模板
│   └── ui/                 # UI 组件与交互
├── scripts/
│   ├── build.sh            # 本地构建脚本
│   ├── build-and-verify.sh # 打包验证脚本
│   ├── install-hooks.sh    # Git hooks 安装
│   └── git-hooks/          # Git hooks 脚本
├── .github/workflows/      # CI/CD 配置
└── dist/                   # 编译产物
```

---

## 🗺️ 后续计划

- [ ] 丰富命令描述字段（环境变量、工作目录、平台约束等）
- [ ] 提供 Schema 校验
- [ ] 插件示例（执行日志、结果上报等）
- [ ] 补充测试与 CI 流程
- [ ] 多平台行为一致性验证

---

## 📄 许可证

MIT License

---

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

**开发流程**：
1. Fork 本仓库
2. 克隆并安装 Git hooks：`./scripts/install-hooks.sh`
3. 创建特性分支：`git checkout -b feature/your-feature`
4. 提交改动（会自动运行代码检查）
5. 推送到你的 Fork：`git push origin feature/your-feature`
6. 创建 Pull Request

---

## 📮 联系方式

- **Issues**: https://github.com/alpenl/alpen-cli/issues
- **Discussions**: https://github.com/alpenl/alpen-cli/discussions
