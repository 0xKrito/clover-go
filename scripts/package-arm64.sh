#!/usr/bin/env bash
set -euo pipefail

TARGET="${1:-app}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_DIR="${ROOT_DIR}/build"
APP_NAME="CloverGo.app"
APP_DIR="${BUILD_DIR}/${APP_NAME}"
DMG_NAME="CloverGo-1.1.0-arm64.dmg"
DMG_PATH="${BUILD_DIR}/${DMG_NAME}"

echo "=================================================="
echo "  CloverGo (Pure Go + MyGo Metal GPU) 打包"
echo "  Target: ${TARGET}"
echo "=================================================="

# 1. 编译纯 Go 二进制
echo "[1] 编译纯 Go 二进制 (GOOS=darwin GOARCH=arm64 CGO_ENABLED=0)..."
mkdir -p "${BUILD_DIR}"
cd "${ROOT_DIR}"
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o "${BUILD_DIR}/CloverGo" ./cmd/clover

# 1.5 预编译 Swift 原生能力 helper（PDF/水印/Finder sfl3）
echo "[1.5] 预编译 clover-helper (swiftc)..."
swiftc -O -o "${BUILD_DIR}/clover-helper" "${ROOT_DIR}/tools/clover-helper/main.swift"

# 2. 组装 macOS Bundle
echo "[2] 组装 .app 包结构..."
rm -rf "${APP_DIR}"
mkdir -p "${APP_DIR}/Contents/MacOS"
mkdir -p "${APP_DIR}/Contents/Resources"

cp "${BUILD_DIR}/CloverGo" "${APP_DIR}/Contents/MacOS/CloverGo"
chmod +x "${APP_DIR}/Contents/MacOS/CloverGo"
cp "${BUILD_DIR}/clover-helper" "${APP_DIR}/Contents/MacOS/clover-helper"
chmod +x "${APP_DIR}/Contents/MacOS/clover-helper"

# 复制应用图标（优先使用 assets/AppIcon.icns 官方高清拟物图标）
if [ -f "${ROOT_DIR}/assets/AppIcon.icns" ]; then
    cp "${ROOT_DIR}/assets/AppIcon.icns" "${APP_DIR}/Contents/Resources/AppIcon.icns"
    cp "${ROOT_DIR}/assets/AppIcon.icns" "${BUILD_DIR}/AppIcon.icns"
elif [ -f "${BUILD_DIR}/AppIcon.icns" ]; then
    cp "${BUILD_DIR}/AppIcon.icns" "${APP_DIR}/Contents/Resources/AppIcon.icns"
fi

cat <<EOF > "${APP_DIR}/Contents/Info.plist"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDisplayName</key>
    <string>CloverGo</string>
    <key>CFBundleExecutable</key>
    <string>CloverGo</string>
    <key>CFBundleIdentifier</key>
    <string>com.clover.app</string>
    <key>CFBundleName</key>
    <string>CloverGo</string>
    <key>CFBundleIconFile</key>
    <string>AppIcon</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>1.1.0</string>
    <key>CFBundleVersion</key>
    <string>110</string>
    <key>LSMinimumSystemVersion</key>
    <string>12.0</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>CFBundleURLTypes</key>
    <array>
        <dict>
            <key>CFBundleURLName</key>
            <string>CloverGo URL</string>
            <key>CFBundleURLSchemes</key>
            <array>
                <string>fclover</string>
            </array>
        </dict>
    </array>
</dict>
</plist>
EOF

# 3. Ad-Hoc 签名
echo "[3] 执行 macOS xattr 清理与 Ad-Hoc 签名..."
xattr -cr "${APP_DIR}" || true
codesign --force --deep --sign - "${APP_DIR}"
codesign -v "${APP_DIR}"

echo "✅ CloverGo.app 构建完成: ${APP_DIR}"

if [ "${TARGET}" = "dmg" ]; then
    echo "[4] 生成可分发 DMG 镜像..."
    rm -f "${DMG_PATH}"
    DMG_STAGING="${BUILD_DIR}/dmg_staging"
    rm -rf "${DMG_STAGING}"
    mkdir -p "${DMG_STAGING}"

    cp -R "${APP_DIR}" "${DMG_STAGING}/"
    ln -s /Applications "${DMG_STAGING}/Applications"

    hdiutil create -volname "CloverGo" -srcfolder "${DMG_STAGING}" -ov -format UDZO "${DMG_PATH}"
    rm -rf "${DMG_STAGING}"
    echo "✅ DMG 镜像生成完成: ${DMG_PATH}"
fi
