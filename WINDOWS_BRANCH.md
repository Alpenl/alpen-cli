# Windows 分支构建说明

## 🎯 分支用途

`windows-support` 分支专门用于构建 Windows 版本的 alpen-cli。

- ✅ 独立维护，不合并到 main
- ✅ 专用的 Windows 构建流程
- ✅ 生成 `.exe` 可执行文件

## 📦 构建产物

### 自动构建

每次推送到 `windows-support` 分支时，GitHub Actions 会自动构建：

1. **alpen.exe** - Windows 可执行文件
2. **alpen-cli-VERSION-windows-amd64.zip** - 发布包（包含 exe + 文档）

### 查看构建结果

访问：https://github.com/Alpenl/alpen-cli/actions/workflows/windows-build.yml

## 🏷️ 发布新版本

### 创建 Windows 版本标签

使用 `vw` 前缀标记 Windows 版本：

```bash
# 1. 确保在 windows-support 分支
git checkout windows-support

# 2. 创建 Windows 版本标签
git tag -a vw0.3.0 -m "Windows 版本 v0.3.0

- Windows 平台原生支持
- 环境变量自动转换 (export → set)
- 剪贴板功能
- cmd.exe 原生执行"

# 3. 推送标签
git push origin vw0.3.0
```

### 自动发布流程

推送 `vw*` 标签后：

1. ✅ 触发 GitHub Action 构建
2. ✅ 自动运行质量检查
3. ✅ 在 Windows 环境构建 exe
4. ✅ 创建 GitHub Release
5. ✅ 上传 `alpen.exe` 和 ZIP 包

## 📥 用户下载

用户可以从 Release 页面下载：

https://github.com/Alpenl/alpen-cli/releases

### 安装步骤

1. 下载 `alpen.exe`
2. 放到任意目录（如 `C:\Program Files\alpen\`）
3. 添加到 PATH 环境变量
4. 打开 CMD 运行：`alpen --version`

## 🔄 版本命名规范

| 平台 | 标签格式 | 示例 | Release 名称 |
|------|---------|------|-------------|
| Linux/macOS | `v*` | `v0.3.0` | Debian 包 |
| Windows | `vw*` | `vw0.3.0` | Windows exe |

## 🛠️ 手动触发构建

在 GitHub Actions 页面可以手动触发：

1. 访问：https://github.com/Alpenl/alpen-cli/actions/workflows/windows-build.yml
2. 点击 "Run workflow"
3. 选择 `windows-support` 分支
4. 点击 "Run workflow" 按钮

## 📊 构建矩阵

| 项目 | Linux/macOS | Windows |
|------|-------------|---------|
| 分支 | `main` | `windows-support` |
| Workflow | `pipeline.yml` | `windows-build.yml` |
| 运行环境 | `ubuntu-22.04` | `windows-latest` |
| 标签前缀 | `v*` | `vw*` |
| 产物格式 | `.deb` | `.exe` |
| Shell | `/bin/sh` | `cmd.exe` |
| 环境变量 | `export` | `set` |

## 🔍 故障排查

### 构建失败

1. 查看 Actions 日志
2. 检查 Go 版本兼容性
3. 验证依赖是否完整

### 测试构建

本地测试（Windows 环境）：

```powershell
# 构建
go build -o alpen.exe .

# 测试
.\alpen.exe --version
.\alpen.exe ls
```

跨平台编译（Linux 上构建 Windows 版本）：

```bash
GOOS=windows GOARCH=amd64 go build -o alpen.exe .
```

## 📝 维护流程

### 同步主分支的重要修复

如果 main 分支有重要 bugfix：

```bash
# 1. 在 windows-support 分支
git checkout windows-support

# 2. Cherry-pick 特定提交
git cherry-pick <commit-hash>

# 3. 推送
git push origin windows-support
```

### 更新 Windows 特定代码

只在 `windows-support` 分支修改：

```bash
git checkout windows-support
# 修改代码
git add .
git commit -m "Windows: 修复描述"
git push origin windows-support
```

## 🚀 快速发布检查清单

- [ ] 代码已提交到 windows-support 分支
- [ ] 所有测试通过
- [ ] 更新了 WINDOWS_SUPPORT.md（如需要）
- [ ] 创建 vw* 标签
- [ ] 推送标签到远程
- [ ] 等待 GitHub Actions 完成
- [ ] 验证 Release 页面的文件
- [ ] 测试下载的 exe 文件

---

**Windows 分支由 v0.3.0 开始维护** ✨
