#!/usr/bin/env bash
# ==============================================================================
# CloverGo 演示沙箱与零隐私截图素材生成器
# 用于开源宣传截图、GIF 录制与功能演示，彻底杜绝个人文件名与私密路径外泄。
# ==============================================================================
set -euo pipefail

DEMO_ROOT="${1:-/tmp/CloverDemo}"

echo "=================================================="
echo "  正在创建 CloverGo 演示沙箱: ${DEMO_ROOT}"
echo "=================================================="

rm -rf "${DEMO_ROOT}"
mkdir -p "${DEMO_ROOT}"

# 1. 常用目录分类
mkdir -p "${DEMO_ROOT}/Projects/clover-next"
mkdir -p "${DEMO_ROOT}/Documents/Reports"
mkdir -p "${DEMO_ROOT}/Design_Assets/Icons"
mkdir -p "${DEMO_ROOT}/Wallpapers"
mkdir -p "${DEMO_ROOT}/Archives"

# 2. 构造代码项目结构（用于展示树形、代码文件与 Git 感知）
cat <<'EOF' > "${DEMO_ROOT}/Projects/clover-next/package.json"
{
  "name": "clover-next",
  "version": "1.0.0",
  "description": "Next-generation native file management frontend",
  "main": "index.ts",
  "scripts": {
    "dev": "vite",
    "build": "tsc && vite build"
  },
  "dependencies": {
    "react": "^19.0.0",
    "react-dom": "^19.0.0"
  }
}
EOF

cat <<'EOF' > "${DEMO_ROOT}/Projects/clover-next/App.tsx"
import React, { useState } from 'react';

export function Dashboard() {
  const [activeTab, setActiveTab] = useState('overview');
  return (
    <div className="container mx-auto p-6">
      <header className="flex justify-between items-center mb-8">
        <h1 className="text-2xl font-bold tracking-tight">Clover Workspace</h1>
        <span className="badge badge-accent">v1.0.0</span>
      </header>
    </div>
  );
}
EOF

# 3. 构造丰富文档与 Markdown 文件（用于展示原生富文本渲染/源码双模）
cat <<'EOF' > "${DEMO_ROOT}/Documents/README.md"
# 🍀 CloverGo Overview

CloverGo is a high-performance, lightweight file manager for macOS built with **Go 1.27** and **Metal GPU acceleration**.

### Key Highlights
- **Ultra-low Footprint**: ~35MB RAM, <16MB Single Binary
- **Equal Dual-Pane**: Side-by-side effortless drag and drop
- **Native Markdown Preview**: Switch between rendered HTML and source view
- **Archive VFS**: In-place zero-extraction browsing for `.zip` and `.tar.gz`

> Designed with Apple Human Interface Guidelines in mind.
EOF

cat <<'EOF' > "${DEMO_ROOT}/Documents/Reports/Quarterly_Review_2026.txt"
Q4 Performance Benchmark Summary
--------------------------------
- Idle CPU Usage: 0.0%
- Cold Startup: < 120ms
- Memory (RSS): 34.2 MB
- Frame Rate: Solid 120 FPS (ProMotion)
EOF

# 4. 构造只读免解压压缩包（用于展示 Archive VFS 穿透）
TEMP_BUNDLE="/tmp/clover_bundle_staging"
rm -rf "${TEMP_BUNDLE}" && mkdir -p "${TEMP_BUNDLE}/bundle-v1"/{core,specs}
echo "Sample config" > "${TEMP_BUNDLE}/bundle-v1/core/config.json"
echo "Architecture diagram payload" > "${TEMP_BUNDLE}/bundle-v1/specs/architecture.txt"
ditto -c -k "${TEMP_BUNDLE}" "${DEMO_ROOT}/Archives/release-bundle-2026.zip"
rm -rf "${TEMP_BUNDLE}"

# 5. 复制系统精美壁纸样本（用于展示图片超级预览与缩略图）
FOUND_WALLPAPER=0
for wp in /System/Library/Desktop\ Pictures/*.heic /Library/Desktop\ Pictures/*.heic; do
  if [ -f "$wp" ]; then
    cp "$wp" "${DEMO_ROOT}/Wallpapers/Monterey_Scenic.heic" 2>/dev/null || true
    FOUND_WALLPAPER=1
    break
  fi
done

if [ "$FOUND_WALLPAPER" -eq 0 ]; then
  # 若无权限读取系统壁纸，写入带占位数据的 SVG 矢量图
  cat <<'EOF' > "${DEMO_ROOT}/Wallpapers/Vector_Landscape.svg"
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 500">
  <rect width="800" height="500" fill="#2d3748"/>
  <circle cx="400" cy="250" r="120" fill="#4fd1c5" opacity="0.8"/>
  <text x="400" y="260" font-family="sans-serif" font-size="28" fill="#ffffff" text-anchor="middle">Clover Visual Demo</text>
</svg>
EOF
fi

chmod -R 755 "${DEMO_ROOT}"

echo "=================================================="
echo "✅ 演示沙箱已生成: ${DEMO_ROOT}"
echo "👉 启动演示命令:"
echo "   CGO_ENABLED=0 go run ./cmd/clover ${DEMO_ROOT}"
echo "=================================================="
