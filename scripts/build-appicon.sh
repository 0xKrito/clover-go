#!/usr/bin/env bash
# ==============================================================================
# 将 1024x1024 高清 PNG 一键转换为 macOS 标准 AppIcon.iconset 与 AppIcon.icns
# ==============================================================================
set -euo pipefail

SRC_PNG="${1:-assets/clover_icon_1024.png}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ICONSET_DIR="${ROOT_DIR}/assets/AppIcon.iconset"
ICNS_FILE="${ROOT_DIR}/assets/AppIcon.icns"

if [ ! -f "${SRC_PNG}" ]; then
  echo "❌ 找不到源文件: ${SRC_PNG}"
  exit 1
fi

echo "=================================================="
echo "  生成 macOS 标准高清图标集: ${SRC_PNG}"
echo "=================================================="

rm -rf "${ICONSET_DIR}"
mkdir -p "${ICONSET_DIR}"

# 按照 Apple HIG 标准尺寸批量切图
sips -z 16 16     "${SRC_PNG}" --out "${ICONSET_DIR}/icon_16x16.png" >/dev/null
sips -z 32 32     "${SRC_PNG}" --out "${ICONSET_DIR}/icon_16x16@2x.png" >/dev/null
sips -z 32 32     "${SRC_PNG}" --out "${ICONSET_DIR}/icon_32x32.png" >/dev/null
sips -z 64 64     "${SRC_PNG}" --out "${ICONSET_DIR}/icon_32x32@2x.png" >/dev/null
sips -z 64 64     "${SRC_PNG}" --out "${ICONSET_DIR}/icon_64x64.png" >/dev/null
sips -z 128 128   "${SRC_PNG}" --out "${ICONSET_DIR}/icon_64x64@2x.png" >/dev/null
sips -z 128 128   "${SRC_PNG}" --out "${ICONSET_DIR}/icon_128x128.png" >/dev/null
sips -z 256 256   "${SRC_PNG}" --out "${ICONSET_DIR}/icon_128x128@2x.png" >/dev/null
sips -z 256 256   "${SRC_PNG}" --out "${ICONSET_DIR}/icon_256x256.png" >/dev/null
sips -z 512 512   "${SRC_PNG}" --out "${ICONSET_DIR}/icon_256x256@2x.png" >/dev/null
sips -z 512 512   "${SRC_PNG}" --out "${ICONSET_DIR}/icon_512x512.png" >/dev/null
sips -z 1024 1024 "${SRC_PNG}" --out "${ICONSET_DIR}/icon_512x512@2x.png" >/dev/null

# 复制一份作为 README 展示使用的 512x512 PNG
sips -z 512 512 "${SRC_PNG}" --out "${ROOT_DIR}/assets/app-icon.png" >/dev/null

# 调用系统 iconutil 编译为 .icns 二进制资源
iconutil -c icns "${ICONSET_DIR}" -o "${ICNS_FILE}"

# 同步到 build 目录
mkdir -p "${ROOT_DIR}/build"
cp "${ICNS_FILE}" "${ROOT_DIR}/build/AppIcon.icns"

echo "✅ AppIcon.icns 生成成功: ${ICNS_FILE}"
echo "✅ app-icon.png 已更新: ${ROOT_DIR}/assets/app-icon.png"
