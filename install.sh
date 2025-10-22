#!/usr/bin/env bash
set -euo pipefail

# Alpen CLI 一键安装脚本
# 支持国内环境（自动使用代理镜像）
#
# 用法:
#   在线安装: curl -fsSL https://raw.githubusercontent.com/alpenl/alpen-cli/main/install.sh | sudo bash
#   本地安装: sudo bash install.sh
#   强制国内镜像: sudo CHINA_MIRROR=1 bash install.sh

REPO="alpenl/alpen-cli"
BINARY_NAME="alpen"
INSTALL_DIR="/usr/bin"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

info() { echo -e "${GREEN}[INFO]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; exit 1; }
debug() { echo -e "${BLUE}[DEBUG]${NC} $*"; }

# 检查依赖
command -v wget >/dev/null 2>&1 || error "未找到 wget，请先安装: sudo apt-get install wget"
command -v dpkg >/dev/null 2>&1 || error "此脚本仅支持 Debian/Ubuntu 系统"

# 检查 root 权限
[[ $EUID -ne 0 ]] && error "请使用 sudo 运行此脚本"

# 检测网络环境（是否需要使用国内镜像）
detect_network() {
    # 允许手动指定使用国内镜像
    if [[ "${CHINA_MIRROR:-0}" == "1" ]]; then
        return 0
    fi

    # 测试 GitHub API 连通性（超时 3 秒）
    if wget --timeout=3 --tries=1 -qO- "https://api.github.com" >/dev/null 2>&1; then
        return 1  # 国外环境，GitHub 可访问
    else
        return 0  # 国内环境，需要使用镜像
    fi
}

# 获取最新版本
get_latest_version() {
    local api_url="$1"

    info "正在获取最新版本..."
    debug "API 地址: ${api_url}"

    LATEST_VERSION=$(wget -qO- "${api_url}" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

    if [[ -z "$LATEST_VERSION" ]]; then
        return 1
    fi

    info "最新版本: ${LATEST_VERSION}"
    return 0
}

# 检测网络环境并选择镜像源
info "🌐 检测网络环境..."
if detect_network; then
    warn "检测到国内环境，使用镜像加速"
    GITHUB_PROXY="https://gh-proxy.com/https://github.com"
    API_PROXY="https://gh-proxy.com/https://api.github.com"
    USE_MIRROR=true
else
    info "使用 GitHub 官方源"
    GITHUB_PROXY="https://github.com"
    API_PROXY="https://api.github.com"
    USE_MIRROR=false
fi

# 获取最新版本（带重试机制）
if ! get_latest_version "${API_PROXY}/repos/${REPO}/releases/latest"; then
    if [[ "$USE_MIRROR" == "true" ]]; then
        warn "镜像 API 失败，尝试官方源..."
        API_PROXY="https://api.github.com"
        get_latest_version "${API_PROXY}/repos/${REPO}/releases/latest" || error "无法获取最新版本，请检查网络连接"
    else
        warn "官方源失败，尝试镜像..."
        API_PROXY="https://gh-proxy.com/https://api.github.com"
        get_latest_version "${API_PROXY}/repos/${REPO}/releases/latest" || error "无法获取最新版本，请检查网络连接"
    fi
fi

# 构造下载 URL（去除版本号中的 v 前缀）
VERSION_NUMBER="${LATEST_VERSION#v}"  # 去除 v 前缀
DEB_FILE="alpen-cli_${VERSION_NUMBER}_amd64.deb"
DOWNLOAD_URL="${GITHUB_PROXY}/${REPO}/releases/download/${LATEST_VERSION}/${DEB_FILE}"

# 下载安装包
info "正在下载 ${DEB_FILE}..."
debug "下载地址: ${DOWNLOAD_URL}"
TMP_DIR=$(mktemp -d)
trap "rm -rf ${TMP_DIR}" EXIT

wget -q --show-progress -O "${TMP_DIR}/${DEB_FILE}" "${DOWNLOAD_URL}" || error "下载失败，请检查网络连接"

# 安装
info "正在安装 alpen-cli..."
dpkg -i "${TMP_DIR}/${DEB_FILE}" 2>/dev/null || {
    warn "检测到依赖问题，正在修复..."
    apt-get install -f -y
}

# 验证安装
if command -v alpen >/dev/null 2>&1; then
    # 优先从 dpkg 获取已安装的版本（最准确）
    INSTALLED_VERSION=$(dpkg -s alpen-cli 2>/dev/null | grep '^Version:' | awk '{print $2}')

    # 如果 dpkg 没有返回版本，尝试从二进制获取
    if [[ -z "$INSTALLED_VERSION" ]]; then
        INSTALLED_VERSION=$(alpen --version 2>/dev/null | grep -oP '\d+\.\d+\.\d+' || alpen -v 2>/dev/null | grep -oP '\d+\.\d+\.\d+' || echo "unknown")
    fi

    info "✅ 安装成功! 版本: ${INSTALLED_VERSION}"
    info "运行 'alpen --help' 查看使用帮助"
else
    error "安装失败，请检查错误信息"
fi
