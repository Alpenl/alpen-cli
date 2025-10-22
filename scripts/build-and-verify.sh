#!/usr/bin/env bash

# 本地 DEB 包构建和验证脚本
# 用于测试打包流程，确保不包含无关文件

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🏗️  开始本地构建测试...${NC}"
echo ""

# 1. 清理旧的构建产物
echo -e "${YELLOW}1️⃣  清理旧的构建产物...${NC}"
rm -rf alpen *.deb alpen-cli-*/ 2>/dev/null || true
echo -e "${GREEN}✓ 清理完成${NC}"
echo ""

# 2. 构建二进制
echo -e "${YELLOW}2️⃣  构建二进制文件...${NC}"
VERSION="0.0.0-test"
go build -trimpath -ldflags="-s -w -X main.version=$VERSION" -o alpen .
chmod +x alpen

# 验证构建
BUILT_VERSION=$(./alpen --version 2>&1 | grep -oP 'v?\d+\.\d+\.\d+' || echo "unknown")
echo -e "${GREEN}✓ 构建完成: ${BUILT_VERSION}${NC}"
echo ""

# 3. 打包为 DEB
echo -e "${YELLOW}3️⃣  打包为 DEB...${NC}"
PKG_DIR="alpen-cli_${VERSION}_amd64"

# 创建包结构
mkdir -p ${PKG_DIR}/DEBIAN
mkdir -p ${PKG_DIR}/usr/bin
mkdir -p ${PKG_DIR}/usr/share/doc/alpen-cli

# 复制二进制文件（唯一的运行时依赖）
cp alpen ${PKG_DIR}/usr/bin/
chmod 755 ${PKG_DIR}/usr/bin/alpen

# 复制文档
if [ -f README.md ]; then
    cp README.md ${PKG_DIR}/usr/share/doc/alpen-cli/
fi

# 创建 copyright 文件
cat > ${PKG_DIR}/usr/share/doc/alpen-cli/copyright <<EOF
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: alpen-cli
Source: https://github.com/alpenl/alpen-cli

Files: *
Copyright: $(date +%Y) alpenl <yangyuyang91@gmail.com>
License: MIT
EOF

# 创建 control 文件
cat > ${PKG_DIR}/DEBIAN/control <<EOF
Package: alpen-cli
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: amd64
Maintainer: alpenl <yangyuyang91@gmail.com>
Description: Alpen CLI - 团队脚本统一管理工具
 配置驱动的命令树架构，通过 YAML 配置自动生成 CLI 命令。
 .
 主要特性:
  - 动态命令注册与别名支持
  - 交互式菜单选择
  - 全局和项目级配置
  - 脚本仓库管理
EOF

echo -e "${GREEN}✓ 包结构创建完成${NC}"
echo ""

# 4. 验证包内容
echo -e "${YELLOW}4️⃣  验证包内容（确保没有无关文件）...${NC}"
echo ""
echo -e "${BLUE}📋 包内文件列表:${NC}"
find ${PKG_DIR} -type f | sort | while read file; do
    size=$(du -h "$file" | cut -f1)
    echo "  $file ($size)"
done
echo ""

# 构建 DEB 包
dpkg-deb --build ${PKG_DIR}

echo -e "${GREEN}✓ DEB 包构建完成${NC}"
echo ""

# 5. 详细验证
echo -e "${YELLOW}5️⃣  详细验证 DEB 包...${NC}"
DEB_FILE="${PKG_DIR}.deb"

if [ -f "$DEB_FILE" ]; then
    echo ""
    echo -e "${BLUE}📦 DEB 包信息:${NC}"
    dpkg-deb --info "$DEB_FILE"

    echo ""
    echo -e "${BLUE}📂 DEB 包内容:${NC}"
    dpkg-deb --contents "$DEB_FILE"

    echo ""
    echo -e "${BLUE}📊 文件大小统计:${NC}"
    DEB_SIZE=$(du -h "$DEB_FILE" | cut -f1)
    BINARY_SIZE=$(du -h alpen | cut -f1)
    echo "  DEB 包总大小: $DEB_SIZE"
    echo "  二进制大小:   $BINARY_SIZE"
    echo ""

    # 检查是否包含不应该存在的文件
    echo -e "${YELLOW}🔍 检查不应该存在的文件...${NC}"
    UNWANTED_PATTERNS=(
        "node_modules"
        "package.json"
        "package-lock.json"
        ".git"
        ".github"
        "docs/"
        "scripts/git-hooks"
        ".env"
        ".mcp.json"
        "config-editor.html"
    )

    FOUND_UNWANTED=0
    for pattern in "${UNWANTED_PATTERNS[@]}"; do
        if dpkg-deb --contents "$DEB_FILE" | grep -q "$pattern"; then
            echo -e "${RED}✗ 发现不应该存在的文件: $pattern${NC}"
            FOUND_UNWANTED=1
        fi
    done

    if [ $FOUND_UNWANTED -eq 0 ]; then
        echo -e "${GREEN}✓ 未发现无关文件${NC}"
    else
        echo -e "${RED}✗ 发现无关文件，请检查打包脚本${NC}"
        exit 1
    fi

    echo ""
    echo -e "${GREEN}✅ 所有检查通过！${NC}"
    echo ""
    echo -e "${BLUE}💡 测试安装命令:${NC}"
    echo "  sudo dpkg -i $DEB_FILE"
    echo "  alpen --version"
    echo ""
else
    echo -e "${RED}✗ DEB 包构建失败${NC}"
    exit 1
fi

echo -e "${BLUE}🧹 清理测试产物...${NC}"
echo "运行以下命令清理:"
echo "  rm -rf alpen *.deb alpen-cli-*/"
echo ""
