#!/usr/bin/env bash

# Git Hooks 安装脚本
# 将版本控制的 hooks 链接到 .git/hooks 目录

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
HOOKS_SOURCE_DIR="$PROJECT_ROOT/scripts/git-hooks"
HOOKS_TARGET_DIR="$PROJECT_ROOT/.git/hooks"

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🔧 安装 Git Hooks...${NC}"
echo ""

# 检查是否在 Git 仓库中
if [ ! -d "$PROJECT_ROOT/.git" ]; then
    echo "错误: 未检测到 .git 目录，请在 Git 仓库根目录运行此脚本"
    exit 1
fi

# 检查 hooks 源目录
if [ ! -d "$HOOKS_SOURCE_DIR" ]; then
    echo "错误: hooks 源目录不存在: $HOOKS_SOURCE_DIR"
    exit 1
fi

# 创建 .git/hooks 目录（如果不存在）
mkdir -p "$HOOKS_TARGET_DIR"

# 安装 hooks
INSTALLED_COUNT=0

for hook_file in "$HOOKS_SOURCE_DIR"/*; do
    if [ -f "$hook_file" ]; then
        hook_name=$(basename "$hook_file")
        target_file="$HOOKS_TARGET_DIR/$hook_name"

        # 备份现有的 hook（如果存在且不是符号链接）
        if [ -f "$target_file" ] && [ ! -L "$target_file" ]; then
            backup_file="$target_file.backup.$(date +%Y%m%d%H%M%S)"
            echo -e "${YELLOW}⚠️  备份现有 hook: $hook_name → $(basename $backup_file)${NC}"
            mv "$target_file" "$backup_file"
        fi

        # 删除旧的符号链接
        if [ -L "$target_file" ]; then
            rm "$target_file"
        fi

        # 创建符号链接
        ln -s "$hook_file" "$target_file"
        echo -e "${GREEN}✓ 安装: $hook_name${NC}"
        INSTALLED_COUNT=$((INSTALLED_COUNT + 1))
    fi
done

echo ""
echo -e "${GREEN}✅ 成功安装 $INSTALLED_COUNT 个 Git Hook(s)${NC}"
echo ""
echo -e "${YELLOW}📋 已安装的 hooks：${NC}"
ls -lh "$HOOKS_TARGET_DIR" | grep -v "sample" | grep "^l" | awk '{print "  - " $9 " → " $11}' || echo "  (无)"
echo ""
echo -e "${BLUE}💡 提示：${NC}"
echo "  - Hooks 已启用，每次 git commit 时会自动运行检查"
echo "  - 如需跳过检查，使用: git commit --no-verify"
echo "  - 如需卸载，删除符号链接: rm .git/hooks/pre-commit"
echo ""
