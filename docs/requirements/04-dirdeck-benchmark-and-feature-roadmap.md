# 04 - DirDeck 竞品深度对标与 CloverGo 关键特性实施路线图 (Benchmark & Roadmap)

> **文档性质**：[事前] 竞品分析、需求对齐与实施路线图  
> **对标产品**：[DirDeck Community 仓库](../other_res/dirdeck-community)（基于官方 14 组高保真功能截图与设计文档）  
> **执行工程**：`CloverGo`（纯 Go 1.27 + `egoist/mygo` + macOS 系统互操作架构）  
> **核心原则**：奥卡姆剃刀定律（如无必要，勿增实体），聚焦高价值体验，剔除冗余悬浮球组件。

---

## 1. 竞品定位与产品心智 (Product DNA)

DirDeck 是一款定位于 **“文件管理界瑞士军刀”** 的现代 macOS 15+ 工作台应用。它的核心竞争力在于：**将现代浏览器管理范式（多标签、书签栏、工作区）与全流程生产力微工具（免解压穿透、就地处理、全局联动、快速搜索）深度融合**。

经对齐 CloverGo 当前核心诉求与工程实现边界，确定锁定 **5 大高价值核心能力域**：
1. **左右对等双栏（Equal Dual-Pane）**：单标签内左右独立窗格，激活窗格蓝色高亮聚焦框，跨栏无缝拖放。
2. **横向多层级书签栏（Hierarchical Bookmark Bar）**：导航栏下方常驻，支持文件夹多级嵌套下拉与快捷直达。
3. **批量重命名可视化弹窗（Batch Rename Modal）**：基于现有后端算法，补全积木式规则链与实时 Before/After 差异预览。
4. **压缩包就地免解压虚拟浏览（In-place Archive VFS）**：`.zip`/`.tar.gz` 虚拟只读穿透展开，告别解压垃圾。
5. **8 大高频右键深度系统联动矩阵**：设为默认打开程序、在新标签页打开、快速查看、重命名、复制绝对路径、在 Finder 中显示、加入书签、超级预览。

> **明确剔除项**：**悬浮球（Floating Orb）**。由于 CloverGo 的树形目录与可点击面包屑已天然具备就近返回上一级与层级跳转能力，恪守极简主义，不引入遮挡内容的悬浮微件。

---

## 2. 竞品优势功能与视觉交互品味拆解

### 2.1 视觉系统与产品品味 (Visual Aesthetics & Polish)

![DirDeck 工作区全景](../other_res/dirdeck-community/assets/screenshots/workspaces.webp)

1. **紧凑的一体化顶栏**：
   - 窗口红绿灯（Traffic Lights）与浏览器式标签页（Tabs）同高度合并，消除了传统 macOS 冗余的空白标题栏。
2. **高对比激活聚焦环（Active Pane Accent Border）**：
   - 在多窗格布局下，当前获得键盘/鼠标焦点的窗格会被 **1.5px 的鲜明蓝色外边框（Accent Ring）** 包覆。彻底解决了多窗格文件管理器中“用户不知道快捷键和粘贴会落在哪个窗格”的认知盲区。
3. **默认应用图标角标（Default App Badges）**：
   - 见截图 `default-app.webp`，在 `.mp4`, `.zip`, `.cer` 等文件图标的右下角，直观叠加系统当前默认关联 App（QuickTime、归档实用工具、钥匙串）的微缩徽标，信息密度极高。

---

### 2.2 多标签页与多层级书签栏体系 (Tabs & Bookmarks)

![从 Finder 导入收藏](../other_res/dirdeck-community/assets/screenshots/finder-import.webp)

1. **双层定位体系**：
   - **标签页（Tabs）**：“留住眼前的工作”。存放当前正在处理的 2~5 个临时任务目录，支持拖拽重排与关闭。
   - **书签栏（Bookmark Bar）**：“留住常去的位置”。紧随 Toolbar 下方，支持文件夹多层级嵌套（如 `桌面`、`dev`、`常用/项目A`），点击弹出下拉菜单快速穿透。
2. **Finder 收藏一键导入（Finder Import）**：
   - 提供扫描 Finder 侧边栏收藏目录的导入弹窗，支持勾选批量导入，免除冷启动迁移成本。

---

### 2.3 左右对等双栏 (Equal Dual-Pane)

1. **左右独立窗格**：
   - 两个窗格各自保留独立的历史栈（`<` `>`）、独立的面包屑导航条、独立的排序方式与文件选中态。
   - 跨窗格拖拽顺畅，大幅简化素材归档、文件整理与目录对比。
2. **工作区状态机（Workspaces）**：
   - 支持将当前打开的标签、分屏与路径配置保存为命名工作区，支持自动保存与一键场景复原。

---

### 2.4 超级预览与压缩包就地穿透 (Super Preview & In-place Archive)

![超级预览与文件夹内容](../other_res/dirdeck-community/assets/screenshots/side-folder.webp)

![压缩包就地穿透浏览](../other_res/dirdeck-community/assets/screenshots/archive-browser.webp)

1. **超级预览（Super Preview）**：
   - 右侧常驻栏除了展示常规文件元数据（体积、创建/修改时间）外，对于文件夹与压缩包，可就地展开包内树形预览，无需双击进入。
2. **压缩包免解压穿透浏览**：
   - 彻底打破“必须先解压到临时目录才能看”的通病。在列表中选中的 `.zip`, `.tar.gz` 等直接被虚拟化为只读文件夹，支持逐层展开与内容预览。

---

### 2.5 瑞士军刀微工具与右键上下文生态

![批量重命名规则链](../other_res/dirdeck-community/assets/screenshots/batch-rename.webp)

![默认应用与高频右键](../other_res/dirdeck-community/assets/screenshots/default-app.webp)

![微工具开关配置](../other_res/dirdeck-community/assets/screenshots/tools-config.webp)

- **积木式批量重命名**：可组合 `前缀`、`后缀`、`查找替换`、`正则表达式`、`数字序列`，右侧实时显示 Before/After 映射与红绿冲突校验灯。
- **高频深度系统右键**：设为默认打开程序、在新标签页打开、快速查看、重命名、复制绝对路径、在 Finder 中显示、加入书签、超级预览。

---

## 3. CloverGo 落地技术方案与工程可行性矩阵 (Feasibility & Cost)

### 3.1 核心特性落地对比与成本清单

| 模块 | 核心能力 | 方案决策 | 实现成本 | 技术路径与物理约束说明 |
| :--- | :--- | :---: | :---: | :--- |
| **布局** | **左右对等双栏 (Equal Dual-Pane)** | **采用** | **约 2 天** | • 数据层：`TabState` 增加 `Panes [2]*PaneState`，`ActivePaneIdx` 标识焦点。<br>• 渲染层：`window_view.go` 使用左右等宽 Flex 容器，非激活窗格边框透明，激活窗格渲染 `AccentBlue` 1.5px 边框。<br>• 交互：`Tab` 键在左右栏切换焦点，文件拖入另一栏触发移动/复制。 |
| **导航** | **横向多层级书签栏 (Bookmark Bar)** | **采用** | **约 1 天** | • 数据层：在 `config.Preferences` 增加 `BookmarkBar []BookmarkItem`（含 `Children`）。<br>• 渲染层：在 Toolbar 与主体之间新增 `BookmarkBarView`（高度 26px），文件夹项点击展开下拉浮层。 |
| **工具** | **批量重命名可视化弹窗 (Batch Rename)** | **采用** | **约 1 天** | • 后端：完全复用已有的 `internal/domain/batch_rename.go`。<br>• 前端：在 `modals_view.go` 编写规则链选择器与双列对比实时预览表格。 |
| **核心** | **压缩包免解压穿透 (Archive VFS)** | **采用** | **约 2~3 天** | • 抽象 `VFS` 路径挂载：当路径为 `/path/to/archive.zip#inside/sub` 时，委托给 Go 原生 `archive/zip` 或 `archive/tar` 流式读取 `fs.DirEntry`。<br>• 状态提示：底栏标明“只读虚拟挂载”，禁用写操作。 |
| **右键** | **8 大高频深度右键联动** | **采用** | **约 1.5 天** | • **设为默认程序**：调用 LaunchServices / `duti` 设置 UTI 关联。<br>• **在新标签页打开**：`coord.Tabs.NewTab(path)`。<br>• **快速查看**：调用已有 `internal/macos/quicklook.go`。<br>• **重命名**：就地内联重命名聚焦。<br>• **复制绝对路径**：写入系统剪贴板（NSPasteboard）。<br>• **在 Finder 中显示**：`exec.Command("open", "-R", path)`。<br>• **加入书签**：追加至用户书签列表。<br>• **超级预览**：在右侧常驻栏快速加载该文件夹/压缩包的内部文件清单与元数据。 |
| **微交互**| **分栏悬浮球 (Floating Orb)** | **剔除** | **0 天** | **奥卡姆剃刀**：CloverGo 面包屑与目录结构已支持就近跳转，悬浮球会遮挡底部文件内容，予以排除。 |

---

## 4. 架构设计与数据流向 (Architecture Blueprint)

### 4.1 左右双栏分层与焦点状态流

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            CloverGo Root Window                             │
├─────────────────────────────────────────────────────────────────────────────┤
│  TitleBar & Tabs: [ Tab 1: /Users/work ] [ Tab 2: /Downloads ]  [ + ]       │
├─────────────────────────────────────────────────────────────────────────────┤
│  Toolbar: [ < ] [ > ] [ ^ ]  [ Breadcrumbs / Path Capsule ]  [ Search ]     │
├─────────────────────────────────────────────────────────────────────────────┤
│  Bookmark Bar: 📁 Desktop  📁 Projects (▾)  📁 Docs  🌐 GitHub              │
├───────────────┬──────────────────────────────┬──────────────────────────────┤
│               │  Left Pane [Active Focus] 🟦  │  Right Pane [Inactive]       │
│               ├──────────────────────────────┼──────────────────────────────┤
│               │ Path: /Users/work/project_a  │ Path: /Volumes/Backup        │
│    Sidebar    ├──────────────────────────────┼──────────────────────────────┤
│  (Collapsible)│ 📄 main.go                   │ 📁 release_202610            │
│               │ 📄 go.mod                    │ 📦 archive.zip [Virtual VFS] │
│               │ 📁 internal                  │ 📄 readme.txt                │
│               │                              │                              │
├───────────────┴──────────────────────────────┴──────────────────────────────┤
│  Status Bar: 12 items (1 selected)                  Free Disk: 142 GB       │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 4.2 8 大右键菜单行为流转矩阵

┌──────────────────────────────┬────────────────────────────────┬───────────────────────────────┐
│ 触发菜单项                   │ 执行逻辑与底层系统调用          │ 视觉反馈与状态变动            │
├──────────────────────────────┼────────────────────────────────┼───────────────────────────────┤
│ 设为默认打开程序             │ LaunchServices API / `duti`    │ 弹出应用选择列表，更新角标    │
│ 在新标签页打开               │ `coord.Tabs.NewTab(path)`      │ 顶栏新增 Tab 并自动聚焦       │
│ 快速查看 (Space)             │ `macos.QuickLookPreview(path)` │ 弹出 macOS 原生 QuickLook 浮窗│
│ 重命名                       │ 激活当前行的 `IsRenaming=true` │ 变为内联可编辑输入框          │
│ 复制绝对路径                 │ 写入系统剪贴板 NSPasteboard    │ 底部状态栏短暂 Toast 提示     │
│ 在 Finder 中显示             │ `open -R <path>`               │ 唤起并置顶 Finder 定位到文件  │
│ 将所选项目加入书签/收藏夹    │ 追加至 `Preferences.Bookmarks` │ 书签栏/侧边栏即时新增图标项   │
│ 超级预览                     │ 右侧预览栏切换为 `InspectMode` │ 展示文件夹层级或压缩包内容树  │
└──────────────────────────────┴────────────────────────────────┴───────────────────────────────┘

---

## 5. 落地三阶段演进路线图 (Evolution Roadmap)

### 阶段一：左右对等双栏 + 横向书签栏 + 8 大右键联动（P0，预计 3~4 工作日）
1. **重构 `TabState`**：支持左右双 `PaneState`，接入 `Tab` 键焦点轮转与高对比蓝色聚焦框。
2. **实现横向书签栏**：增加 Toolbar 下方的多级书签条，支持目录嵌套下拉。
3. **补齐 8 大右键深度系统操作**：
   - 设为默认打开程序
   - 在新标签页打开
   - 快速查看（Space）
   - 重命名
   - 复制绝对路径
   - 在 Finder 中显示
   - 加入书签
   - 超级预览
4. **批量重命名可视化弹窗**：打通 `batch_rename.go` 的前台交互界面与实时对比表格。

### 阶段二：压缩包就地免解压虚拟浏览（P1，预计 2~3 工作日）
1. **抽象虚拟文件系统（VFS）**：
   - 识别 `.zip` 与 `.tar.gz` 后缀，构建基于 `archive/zip` 的内存虚拟目录树。
   - 实现包内文件的只读双击预览与空格 QuickLook。

### 阶段三：视觉高级质感（P2，预计 1~2 工作日）
1. **默认应用微缩角标**：在文件缩略图管线中合成默认关联 App 的小图标。
2. **Finder 收藏一键导入**：读取 macOS 用户配置，一键迁移已有常用目录。
