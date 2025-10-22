#!/usr/bin/env bash
set -euo pipefail

# Alpen CLI 一键安装脚本
# 用法: curl -fsSL https://raw.githubusercontent.com/USER/REPO/main/install.sh | sudo bash

REPO="alpenl/alpen-cli"
BINARY_NAME="alpen"
INSTALL_DIR="/usr/bin"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() { echo -e "${GREEN}[INFO]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }

# 检查依赖
command -v wget >/dev/null 2>&1 || error "未找到 wget，请先安装: sudo apt-get install wget"
command -v dpkg >/dev/null 2>&1 || error "此脚本仅支持 Debian/Ubuntu 系统"

# 检查 root 权限
[[ $EUID -ne 0 ]] && error "请使用 sudo 运行此脚本"

# 获取最新版本
info "正在获取最新版本..."
LATEST_VERSION=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
[[ -z "$LATEST_VERSION" ]] && error "无法获取最新版本，请检查网络连接"

info "最新版本: ${LATEST_VERSION}"

# 构造下载 URL
DEB_FILE="alpen-cli_${LATEST_VERSION}_amd64.deb"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_VERSION}/${DEB_FILE}"

# 下载安装包
info "正在下载 ${DEB_FILE}..."
TMP_DIR=$(mktemp -d)
trap "rm -rf ${TMP_DIR}" EXIT

wget -q --show-progress -O "${TMP_DIR}/${DEB_FILE}" "${DOWNLOAD_URL}" || error "下载失败"

# 安装
info "正在安装 alpen-cli..."
dpkg -i "${TMP_DIR}/${DEB_FILE}" 2>/dev/null || {
    warn "检测到依赖问题，正在修复..."
    apt-get install -f -y
}

# 验证安装
if command -v alpen >/dev/null 2>&1; then
    INSTALLED_VERSION=$(alpen --version 2>/dev/null | grep -oP '\d+\.\d+\.\d+' || alpen -v 2>/dev/null || echo "unknown")
    info "✅ 安装成功! 版本: ${INSTALLED_VERSION}"
    info "运行 'alpen --help' 查看使用帮助"
else
    error "安装失败，请检查错误信息"
fi
