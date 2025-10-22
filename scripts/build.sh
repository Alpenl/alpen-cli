#!/usr/bin/env bash

# 本地构建脚本
# 自动从 git tag 获取版本号并注入到二进制

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🏗️  Alpen CLI 本地构建${NC}"
echo ""

# 获取版本信息
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo -e "${YELLOW}版本信息:${NC}"
echo "  Version: $VERSION"
echo "  Commit:  $COMMIT"
echo "  Date:    $BUILD_DATE"
echo ""

# 构建
echo -e "${YELLOW}开始构建...${NC}"
go build -v \
  -trimpath \
  -ldflags="-s -w \
    -X 'github.com/alpen/alpen-cli/cmd.version=${VERSION}' \
    -X 'github.com/alpen/alpen-cli/cmd.commit=${COMMIT}' \
    -X 'github.com/alpen/alpen-cli/cmd.date=${BUILD_DATE}'" \
  -o alpen \
  .

chmod +x alpen

echo ""
echo -e "${GREEN}✅ 构建完成${NC}"
echo ""

# 验证
echo -e "${BLUE}📋 验证版本信息:${NC}"
./alpen --version

echo ""
echo -e "${GREEN}💡 运行 './alpen --help' 查看帮助${NC}"
