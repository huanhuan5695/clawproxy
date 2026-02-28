#!/usr/bin/env bash

set -euo pipefail

APP_NAME="clawproxy"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OUTPUT_DIR="$PROJECT_ROOT/dist"
TARGET_OS="${1:-}"
TARGET_ARCH="${2:-}"

print_usage() {
  cat <<USAGE
用法:
  ./scripts/build.sh [windows|mac|linux] [amd64|arm64]

说明:
  - 不传参数时会交互式选择目标平台。
  - 仅传操作系统时会按默认架构构建:
      windows -> amd64
      linux   -> amd64
      mac     -> arm64   (适合你的 Mac M4)

示例:
  ./scripts/build.sh mac
  ./scripts/build.sh windows amd64
  ./scripts/build.sh linux arm64
USAGE
}

pick_target_os() {
  echo "请选择要打包的目标系统:"
  select os in "mac" "linux" "windows" "退出"; do
    case "$os" in
      mac|linux|windows)
        TARGET_OS="$os"
        break
        ;;
      退出)
        echo "已退出。"
        exit 0
        ;;
      *)
        echo "无效选项，请重试。"
        ;;
    esac
  done
}

normalize_os() {
  case "$1" in
    mac|darwin) echo "darwin" ;;
    linux) echo "linux" ;;
    windows|win) echo "windows" ;;
    *)
      echo "不支持的系统: $1" >&2
      print_usage
      exit 1
      ;;
  esac
}

default_arch_for_os() {
  case "$1" in
    darwin) echo "arm64" ;;
    linux|windows) echo "amd64" ;;
  esac
}

validate_arch() {
  case "$1" in
    amd64|arm64) ;;
    *)
      echo "不支持的架构: $1 (仅支持 amd64/arm64)" >&2
      exit 1
      ;;
  esac
}

if [[ "${TARGET_OS}" == "-h" || "${TARGET_OS}" == "--help" ]]; then
  print_usage
  exit 0
fi

if [[ -z "$TARGET_OS" ]]; then
  pick_target_os
fi

GOOS_VALUE="$(normalize_os "$TARGET_OS")"

if [[ -z "$TARGET_ARCH" ]]; then
  GOARCH_VALUE="$(default_arch_for_os "$GOOS_VALUE")"
else
  validate_arch "$TARGET_ARCH"
  GOARCH_VALUE="$TARGET_ARCH"
fi

mkdir -p "$OUTPUT_DIR"

EXT=""
if [[ "$GOOS_VALUE" == "windows" ]]; then
  EXT=".exe"
fi

OUTPUT_FILE="$OUTPUT_DIR/${APP_NAME}-${GOOS_VALUE}-${GOARCH_VALUE}${EXT}"

echo "项目目录: $PROJECT_ROOT"
echo "开始构建: GOOS=$GOOS_VALUE GOARCH=$GOARCH_VALUE"
echo "输出文件: $OUTPUT_FILE"

(
  cd "$PROJECT_ROOT"
  CGO_ENABLED=0 GOOS="$GOOS_VALUE" GOARCH="$GOARCH_VALUE" \
    go build -o "$OUTPUT_FILE" .
)

echo "构建完成: $OUTPUT_FILE"
