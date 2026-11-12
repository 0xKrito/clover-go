# CloverGo

<p align="center">
  <img src="assets/app-icon.png" width="128" height="128" alt="CloverGo App Icon" />
</p>

<p align="center">
  <strong>看起来是克制优雅的 macOS 原生访达，用起来是行云流水的 Windows Clover 多标签，骨子里是 100% 纯 Go 与 Metal GPU 驱动的极速轻量引擎。</strong>
</p>

<p align="center">
  <a href="https://github.com/bighaidao/clover-go/releases"><img src="https://img.shields.io/github/v/release/bighaidao/clover-go?color=34C759&label=Release" alt="Release"></a>
  <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.27+-00ADD8?logo=go" alt="Go Version"></a>
  <a href="https://apple.com/macos"><img src="https://img.shields.io/badge/macOS-12.0+-000000?logo=apple" alt="macOS"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
  <a href="#"><img src="https://img.shields.io/badge/Binary-<16MB-orange" alt="Binary Size"></a>
  <a href="#"><img src="https://img.shields.io/badge/Memory-~35MB-success" alt="Memory Footprint"></a>
</p>

<p align="center">
  <img src="assets/screenshots/overview.png" alt="CloverGo 界面全景展示" width="900" />
</p>

---

## 🌟 为什么选择 CloverGo？(Why CloverGo?)

长久以来，macOS 上的文件管理体验总在两个极端之间撕裂：
1. **系统原生访达 (Finder)**：优雅克制，但缺乏真正的多标签工作流（无法固定标签、无撤销恢复），没有左右对等双栏，查看压缩包必须先解压，批量重命名与预览能力有限；
2. **第三方商业工具 (QSpace / ForkLift 等)**：功能强大，但闭源收费（￥60~200），功能庞杂臃肿，配置项繁复；
3. **Electron 类新型应用**：现代好看，但常驻内存动辄 300MB~600MB，冷启动缓慢，耗电严重。

**CloverGo** 另辟蹊径——基于 **纯 Go (`CGO_ENABLED=0`) + Metal GPU 原生渲染**，打造出一款**单二进制 <16MB、常驻内存仅 ~35MB、0% 空闲 CPU、120Hz 顺滑响应**、且 **100% 纯本地运行、零隐私数据收集** 的现代化 macOS 文件管理器。

### 📊 核心维度对比矩阵

| 维度 / 特性 | macOS 原生访达 (Finder) | 常见商业文件管理器 | CloverGo (开源免费) |
| :--- | :---: | :---: | :---: |
| **标签页体验** | 基础标签（易误关/无法固定） | 支持，但界面偏重 | **Clover 式页感标签、固定标签 (Pin)、关闭撤销恢复 (`⇧⌘T`)** |
| **窗格布局** | 仅单窗格 / 需开多个窗口 | 双栏 / 四栏 | **左右对等双栏 (Equal Dual-Pane)，跨栏拖拽对齐** |
| **Markdown 预览** | 仅纯文本代码或需插件 | 需安装扩展插件 | **内置富文本可视化渲染 / 源码编辑双模秒级切换，支持 GitHub Alerts 与相对路径联动** |
| **压缩包浏览** | 必须双击解压到磁盘 | 部分支持 | **就地免解压虚拟穿透浏览 (VFS)，直接读取 `.zip` / `.tar.gz`** |
| **生产力工具** | 基础批量重命名 | 需外挂辅助工具 | **内置积木式批量重命名、PDF 水印与合并/拆分工具箱** |
| **内存与能耗** | ~80MB 常驻 | ~180MB - 450MB | **~35MB 超低常驻，0% 空闲 CPU，瞬时冷启动** |
| **渲染引擎** | Cocoa AppKit | AppKit / Qt / Web | **Metal GPU 硬件加速 (120Hz ProMotion 瞬时响应)** |
| **开源与隐私** | 闭源系统内置 | 闭源商业软件 | **100% 源码开放 (MIT)，纯本地运行，零遥测上报** |

---

## 🚀 核心杀手级特性 (Feature Highlights)

### 1. 灵魂多标签与工作区管理 (Clover-style Tabs & Workspaces)
- **类似浏览器的标签栏**：新建标签 (`⌘T`)、快速关闭 (`⌘W`)、关闭历史栈一键撤回恢复 (`⇧⌘T`)。
- **固定标签 (Pin Tab)**：右键固定高频目录为紧凑图标，免疫 `⌘W` 误关。
- **同名目录智能消歧**：打开两个相同名称的项目目录（如 `src`）时，自动显示父级目录副标题（如 `src — projectA`）。
- **命名工作区 (Workspaces)**：一键保存当前整套标签页与双栏布局，按工作场景秒级复原。

### 2. 随处可拖拽的双栏工作台 (Anywhere DnD & Dual-Pane)
- **左右对等双栏**：单标签内一键切分左右对等独立窗格，当前获得焦点的窗格呈现鲜明 Accent 高亮环。
- **全场景随处拖放 (Anywhere Drag & Drop)**：
  - 跨栏移动 / 按住 `⌥` 快速复制；
  - 拖拽任意文件悬停至子文件夹行直接移入；
  - 拖入顶部面包屑导航的任意上级胶囊直接返回上级归档；
  - 拖入顶部其他标签页直接归类；
  - 拖拽文件夹至左侧边栏即刻固定为收藏书签；
  - 原生支持与外部桌面、原生 Finder、微信、VSCode 双向拖拽。

### 3. 超级常驻预览系统 (Super Preview & Dual-Mode Markdown)
- **Markdown 原生双模预览**：
  - **渲染预览模式**：内置 Metal 原生富文本渲染引擎，排版精美，告别干瘪纯文本代码；原生支持 **GitHub Alerts 五色告警卡片** (`[!NOTE]` / `[!TIP]` / `[!IMPORTANT]` / `[!WARNING]` / `[!CAUTION]`)，以及 **Markdown 相对路径双向联动**（鼠标悬浮预览目标文件大小与修改时间，点击直接跳转对应目录或定位源码文件）；
  - **源码查看模式**：基于 Zed 内核代码高亮编辑器，带行号与语法高亮，一键无缝切换。
- **纯文本与代码预览**：支持 `PreserveSpaces` 真实缩进保留，代码对齐不走样。
- **多媒体即时预览**：图片高保真缩略图与 Exif 尺寸、PDF 多页微缩图、GIF 动图与视频画框。
- **原生空格 Quick Look**：选中任意项敲击 `Space` 即刻唤起 macOS 原生系统级快速查看。

<p align="center">
  <img src="assets/screenshots/markdown-preview.png" alt="CloverGo Markdown 富文本渲染与原生 QuickLook 预览" width="900" />
</p>

### 4. 压缩包免解压穿透浏览 (In-place Archive VFS)
- 彻底告别“看个压缩包还要先解压出几千个碎片文件”的痛点。
- 双击 `.zip` 或 `.tar.gz` 即可像普通文件夹一样即时穿透进入只读虚拟文件系统 (VFS)，支持逐层深入、文件预览与快速提取。

### 5. 生产力瑞士军刀工具箱 (Productivity Micro-tools)
- **积木式批量重命名 (`⇧⌘R`)**：自由组合序号递增、添加前缀/后缀、查找替换与正则，提供高保真实时 Before/After 对比预览与冲突告警。
- **PDF 瑞士军刀**：支持**多选 PDF 合并**、**页面提取/拆分**以及 **PDF 批量转高清图片**，本地极速转换无隐私外泄。
- **图片防伪与版权水印**：支持单页/全页一键添加倾斜平铺文字水印，透明度与字号可调，离线本地处理。

### 6. macOS 深度原生联动 (Deep macOS Integration)
- **HIG 富原生上下文菜单**：右键菜单深度遵从 Apple HIG 原生规范，支持右侧快捷键对齐、7 色 SVG 实心标签圆点以及废纸篓破坏性红色高亮。
- **侧边栏精致质感**：侧边栏分组支持 `Sticky` 吸顶效果，窗口失焦时自动退饱和 (`InactiveAccent`)，保持与系统原生 App 一致的沉浸感。
- **权限与位置无缝导入**：提供 Finder 常用位置一键导入，搭配 `permissionflow` 完全磁盘访问权限可视化引导，开箱即用。
- **Finder 7 色标签双向互通**：基于 macOS 标准 `bplist00` 二进制 plist 与 FinderInfo，在 CloverGo 打标即在访达生效，反之亦然。
- **高频右键极速联动**：一键“在终端中打开”（自动适配 Terminal / iTerm2 / Ghostty / Warp）、“在编辑器中打开”（Cursor / VSCode / Zed）、一键复制 POSIX / 简写路径。

---

## ⌨️ 全键盘流快捷键速查 (Keyboard Shortcuts)

CloverGo 遵循 macOS 原生键盘操作逻辑与高效文件管理流，核心功能全键位可控：

| 分类 | 快捷键 | 功能说明 |
| :--- | :--- | :--- |
| **标签与工作区** | `⌘T` | 新建标签页（默认打开当前目录） |
| | `⌘W` | 关闭当前标签页（固定标签自动跳过） |
| | `⇧⌘T` | 撤销关闭，恢复最近关闭的标签页 |
| | `⌘1` ~ `⌘9` | 快速切换至第 1 ~ 9 个标签页 |
| | `⌃Tab` | 在所有已打开标签页之间循环向后切换 |
| **视图与分栏** | `⌥⌘D` / `⌘\` | 开启 / 关闭左右对等双栏分屏 |
| | `Tab` | 在左右分栏之间切换当前焦点 |
| | `⌘I` / `⇧⌘P` | 显示简介 / 开启超级常驻预览栏 |
| | `⇧⌘.` | 切换显示 / 隐藏点开头的隐藏文件 |
| | `⌘R` | 强制重新加载当前目录并清空缓存 |
| **导航与搜索** | `⌘[` / `⌘]` | 历史后退 / 前进 |
| | `⌘↑` | 跳转至上级父目录 |
| | `⌘L` / `⌘G` | 激活路径栏直接编辑 / 跳转路径 |
| | `⌘F` | 聚焦工具栏搜索过滤输入框 |
| **文件操作** | `Space` | macOS 原生快速查看 (Quick Look) |
| | `Enter` | 内联重命名当前选中的文件或文件夹 |
| | `⇧⌘N` | 在当前目录新建文件夹 |
| | `⇧⌘R` | 打开积木式批量重命名工具箱 |
| | `⌥⌘C` | 复制选中项的完整 POSIX 绝对路径 |
| | `⌘⌫` | 将选中的文件或文件夹移至废纸篓 |

---

## 📥 安装与快速上手 (Installation)

### 方式 1：直接下载安装包（推荐）
前往 [GitHub Releases](https://github.com/bighaidao/clover-go/releases) 页面，下载最新的 `CloverGo-x.x.x-arm64.dmg`，双击拖入 `Applications` 即可开始使用。

> [!TIP]
> **macOS 安全提示 (Gatekeeper)**：  
> 本项目为开源软件，尚未加入每年 $99 的商业开发者公证计划。首次打开若提示“无法验证开发者”或“已损坏”，只需打开系统「终端」执行以下一行命令即可解除隔离：
> ```bash
> xattr -cr /Applications/CloverGo.app
> ```

### 方式 2：从源码编译构建

前置条件：
- macOS 12.0+ (Apple Silicon / Intel)
- Go 1.27+
- Xcode Command Line Tools (`xcode-select --install`)

```bash
# 1. 克隆代码仓库
git clone https://github.com/bighaidao/clover-go.git
cd clover-go

# 2. 本地直接运行
make run

# 3. 打包生成独立的 CloverGo.app
make bundle

# 4. 生成当前架构的 DMG 安装镜像
make dmg

# 5. 或同时生成 Apple Silicon 与 Intel 双架构 DMG
make dmg-all
```

---

## ⚡ 性能基准与工程设计 (Architecture & Performance)

CloverGo 坚决奉行**奥卡姆剃刀定律**（如无必要，勿增实体），不引入复杂的跨进程 RPC 或重量级 Webview：

```text
┌─────────────────────────────────────────────────────────────┐
│                 Presentation Layer (Metal GPU)              │
│       mygo/ui Virtualized Table / Grid / Markdown / Editor  │
└──────────────────────────────┬──────────────────────────────┘
                               │ 单向驱动 (严格 120Hz 零渲染帧 I/O)
                               ▼
┌─────────────────────────────────────────────────────────────┐
│                 Application & Coordinator Layer             │
│   TabManager · WindowRegistry · PaneState · DualPaneManager │
└──────────────────────────────┬───────────────────────────────┘
                               │
               ┌───────────────┴───────────────┐
               ▼                               ▼
┌─────────────────────────────┐ ┌─────────────────────────────┐
│    In-place Archive VFS     │ │  macOS Native Interop Layer │
│  zip / tar.gz Stream Reader │ │  QuickLook · Tags · Helper  │
└─────────────────────────────┘ └─────────────────────────────┘
```

- **MyGo v0.8.3 引擎强力驱动**：
  - **拖拽过程纯内存态实时预览**：拖拽交互中不产生任何临时文件，拖拽释放 (`DragEnded`) 瞬间触发单次确定性落盘，取消 (`PressCanceled`) 实行无损状态回滚；
  - **`ListState.HoldPlace` 视口锚定**：动态增删与排序时列表视口零跳动，保持丝滑阅读与导航手感；
  - **32MB 圆角遮罩显存复用预算**：图标与缩略图渲染共享固定纹理缓存池，彻底杜绝显存膨胀。
- **单二进制自包含**：编译后核心仅单个二进制，无外部动态库依赖；
- **渲染帧零系统调用**：120Hz 渲染函数 `View(c)` 内严禁执行同步 I/O，所有磁盘读取与元数据解析均在后台协程池异步加载；
- **原生内存占用**：常驻内存稳定在 ~35MB，无任何垃圾堆积。

---

## 🗺️ 跨平台支持路线图 (Platform Roadmap)

- [x] **macOS (Apple Silicon & Intel)**：原生 Metal GPU 加速、FSEvents 实时监听、Finder 7 色标签、QuickLook、原生 PDF 微工具箱。
- [ ] **Windows 跨平台支持**：规划中（预计 v2.0）。基于 MyGo 跨平台 GPU 后端（Direct3D 11）与 Windows Shell 抽象层，无需依赖重型 WebView。

---

## 🤝 参与贡献 (Contributing)

欢迎任何形式的贡献！无论是新特性建议、Bug 反馈还是 Pull Request：
1. Fork 本仓库并新建分支 (`git checkout -b feat/your-feature`)；
2. 保持代码风格规范并通过自测 (`make test`)；
3. 提交修改并发起 Pull Request。

---

## 📄 开源许可证 (License)

本项目基于 [MIT 许可证](LICENSE) 开源。代码开源透明，纯本地安全运行，完全免费商用与自由定制。
