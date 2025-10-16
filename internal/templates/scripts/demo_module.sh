#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
  inspect)
    echo "[demo.conf] 当前脚本路径: $0"
    echo "[demo.conf] 传入参数数量: $#"
    ;;
  *)
    echo "来自 config/demo.conf 的模块化命令示例"
    echo "使用 alpen env 或 alpen ls 可查看聚合后的命令结构"
    ;;
esac
