# 01 - CloverGo 全栈纯 Go 系统架构与拓扑设计 (RFC)

> **架构铁律**：`CGO_ENABLED=0` 纯 Go 单二进制、单向无环依赖、渲染帧 (`View`) 100% 纯函数零 I/O、资源创建与销毁严格对称。

---

## 一、 事中：三大核心架构可视化拓扑图 (Standard Unicode Topologies)

### 1.1 分层与边界拓扑图（单向依赖与物理隔离）

```text
┌────────────────────────────────────────────────────────────────────────────────────────┐
│  Presentation Layer (`internal/ui` — MyGo Metal GPU 立即模式渲染层, 60/120Hz)           │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐  │
│  │ WindowView   │ │ TabBar &     │ │ FileList &   │ │ PreviewPane  │ │ Modals &     │  │
│  │ (Shell/Drag) │ │ Breadcrumb   │ │ IconGrid     │ │ (Code/Image) │ │ Theme/Style  │  │
│  └──────┬───────┘ └──────┬───────┘ └──────┬───────┘ └──────┬───────┘ └──────┬───────┘  │
│         └────────────────┴────────────────┼────────────────┴────────────────┘          │
│                                           │ 只读快照渲染 / 异步 Action 派发 (严禁直接 I/O)│
└───────────────────────────────────────────┼────────────────────────────────────────────┘
                                            ▼
┌────────────────────────────────────────────────────────────────────────────────────────┐
│  Application & Domain Layer (`internal/app` & `internal/domain` — 核心业务与状态编排)   │
│  ┌─────────────────────────┐    ┌─────────────────────────┐    ┌────────────────────┐  │
│  │ WindowRegistry          │───►│ BrowserCoordinator      │───►│ Pure Domain Rules  │  │
│  │ (多窗生命周期/标签拆合) │    │ (TabManager/Selection/  │    │ (FileSorter/Filter/│  │
│  │                         │    │  History/ConflictState) │    │  BatchRename/Path) │  │
│  └─────────────────────────┘    └────────────┬────────────┘    └────────────────────┘  │
└──────────────────────────────────────────────┼─────────────────────────────────────────┘
                                               │ 接口契约调用 (Context + Cancel + Gen)
                                               ▼
┌────────────────────────────────────────────────────────────────────────────────────────┐
│  Infrastructure Layer (`internal/fs`, `icon`, `search`, `macos`, `store` — 外部 I/O)   │
│  ┌────────────────┐ ┌────────────────┐ ┌────────────────┐ ┌─────────────────────────┐  │
│  │ FS & Archive   │ │ Search Engine  │ │ Icon & Thumb   │ │ macOS Native Bridge     │  │
│  │ (ReadDir分页/  │ │ (WalkDir并发/  │ │ Pipeline (LRU+ │ │ (FSEvents/xattr ColorTag│  │
│  │  Copy/Move/Zip)│ │  TTL索引评分)  │ │  Disk缓存队列) │ │  /QuickLook/Trash/URL)  │  │
│  └────────────────┘ └────────────────┘ └────────────────┘ └─────────────────────────┘  │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 数据流向与权威真理源图（Authoritative Store vs Derived State）

```text
  [磁盘文件系统 (APFS)]                 [~/.clover-go/ 配置沙箱]
   │                ▲                     ├─ preferences.json (主题/排序/视图默认项)
   │ 1. 异步分页读  │ 2. Copy/Move/       ├─ workspaces.json  (书签/收藏夹/命名工作区)
   │  & FSEvents    │    Rename/xattr     └─ session.json     (窗口与活动标签恢复快照)
   ▼                │                                   │
┌───────────────────┴───────────────────┐               │ Temp + Rename 原子写
│     Infrastructure Workers (后台协程)  │◄──────────────┘
└───────────────────┬───────────────────┘
                    │ 3. 携带 `loadGen` 经 `mygo.RunOnMain` 投递
                    ▼
┌────────────────────────────────────────────────────────────────────────────────────────┐
│ ★ Authoritative Store (唯一权威内存真理源 — 仅在 UI 主线程读写)                        │
│  • WindowSession.Tabs[] (`TabState`: CurrentPath, RawEntries[], SelectedKeys, History) │
│  • AppStore.Prefs / Favorites / Workspaces / Clipboard (`Paths[]`, `Mode: Copy|Cut`)   │
│  • IconCache.MemoryLRU (`path+mtime+size` ──► `*ui.Bitmap`)                            │
└───────────────────┬────────────────────────────────────────────────────────────────────┘
                    │ 4. 仅在 `RawEntries`、`SortOrder` 或 `FilterQuery` 变化时重算一次
                    ▼
┌────────────────────────────────────────────────────────────────────────────────────────┐
│ ◇ Derived State (只读派生镜像 — 供 120Hz `View(c)` O(1) 直接读取)                      │
│  • `TabState.DisplayEntries[]` (已完成排序、隐藏文件过滤、搜索高亮的最终切片)          │
│  • `TabState.DuplicateSubtitles` (同名标签消歧副标题映射表)                            │
│  • `PreviewState.RenderedSnapshot` (当前选中项的元数据、颜色标签与文本预览切片)        │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

### 1.3 状态与资源生命周期图（确定性销毁 Teardown）

```text
[进程启动 main()]
  │
  ├─► 1. 获取单实例文件锁 `~/.clover-go/gui.lock` (`syscall.Flock`) ──► 进程退出 `defer lock.Release()`
  ├─► 2. 注册 `mygo.App.OnOpenURL` (`fclover://`) 与 `OnOpenFiles`
  │
  └─► [打开窗口 `NewBrowserWindow(seed)`]
        │
        ├─► 创建 `WindowCoordinator` + 注册至 `WindowRegistry`
        │     │
        │     ├─► [切换目录 / 激活 Tab]
        │     │     ├─► 取消旧目录读取 `prevCancel()` (`context.WithCancel`)
        │     │     ├─► 解绑旧目录监听器 `prevWatch.Close()`
        │     │     ├─► 启动新目录 `mygo.Watch(currentPath)` (FSEvents)
        │     │     └─► 启动可视区图标获取任务 (绑定当前 `tab.loadGen`)
        │     │
        │     └─► [关闭 Tab (`CloseTab`)]
        │           ├─► 立即调用 `tab.cancelLoad()` 终止在途搜索/目录计算协程
        │           └─► 将路径快照压入 `ClosedTabs` 栈 (仅留纯文本元数据，释放 `Entries` 切片内存)
        │
        └─► [窗口关闭 `win.OnClose`]
              ├─► 关闭该窗口活跃的 `mygo.Watch` 句柄
              ├─► 取消该窗口所有后台任务 (`coord.Close()`)
              ├─► 从 `WindowRegistry` 注销并触发会话快照原子落盘 (`session.json`)
              └─► 若为最后一个窗口且用户执行 Quit (`⌘Q`)：等待在途文件写入安全中止后退出进程
```

---

## 二、 架构五维审查清单 (Five-Dimensional Architecture Audit)

| 审查维度 | 核心风险点 | 架构级防御与闭环设计 |
| :--- | :--- | :--- |
| **1. 状态归属** | 列表视图与网格视图各自维护选中集合或排序切片，切换视图时状态丢失或漂移。 | **单一真理源**：选中态 (`Selection ui.Selector`)、排序字段 (`SortOrder`) 与过滤后的 `DisplayEntries` 全部归属 `TabState`；切换列表/网格视图仅切换渲染函数，0 状态拷贝。 |
| **2. 生命周期** | 频繁切换目录导致成百上千个 `FSEvents` 句柄泄漏，或旧目录的图标加载挤占新目录带宽。 | 每个窗口仅持有 **1 个活跃目录 Watcher**，切目录或切 Tab 时先 `Close()` 旧 Watcher 再建新句柄；图标请求队列采用 **LIFO + 世代号 (`loadGen`) 校验**，离开目录的任务在出队瞬间直接丢弃。 |
| **3. 异步时序** | 用户在 100ms 内连续点击 `A -> B -> C` 三个文件夹，由于 `A` 是大目录后返回，把 `C` 的文件列表覆盖掉。 | 每次导航递增 `tab.loadGen uint64` 并调用上一次 `cancel()`；后台协程回主线程 (`RunOnMain`) 时强制断言 `if gen != tab.loadGen { return }`，彻底根绝跨异步时序覆盖。 |
| **4. 爆炸半径** | 某个损坏的图片、超长单行巨型日志或无响应的挂载网络盘，导致整个 UI 卡死或 OOM。 | 1. 文本预览强制 `io.LimitReader(f, 64*1024)` 且检测 NUL 字节自动识别二进制文件；<br>2. 目录读取与 `stat` 设置独立 Worker 超时保护；<br>3. 单个文件复制/图标提取报错仅局部降级为默认图标或单条 Toast，绝不 panic 进程。 |
| **5. 单双向门** | 配置文件格式与 Finder 颜色标签 xattr 写入的兼容性。 | 1. **单向门（Finder xattr 颜色标签）**：严格遵循 macOS `com.apple.metadata:_kMDItemUserTags` 标准二进制 plist 数组格式，修改时保留用户非颜色自定义标签，可随时在原生 Finder 中无损读写；<br>2. **双向门（本地缓存与配置）**：统一收敛在 `~/.clover-go/`，删除该目录即可 100% 干净重置。 |

---

## 三、 关键技术方案推演（反敷衍协议：真实工程权衡）

针对迁移过程中的 3 个核心技术决策，对比真实可行的工程方案：

### 决策 1：文件系统与搜索引擎实现方式
- **方案 A（主推）：100% 纯 Go (`CGO_ENABLED=0`) 原生实现 `internal/fs` 与 `internal/search`**
  - **最大获益点**：彻底砍掉 Rust 工具链、Cargo 编译与 C FFI 头文件/JSON 编解码开销；利用 Go 原生 `os.ReadDir`（底层直调 `getdirentries64`，仅取目录项不强制 `stat` 全量扩展属性）+ 并发 Goroutine Worker 池，10 万文件目录首屏可在 `<15ms` 内返回，单二进制编译仅需 `go build` 2 秒完成。
  - **不可承受的代价 (Worst-case Cost)**：需用 Go 重新编写约 400 行目录操作、冲突重命名与搜索评分代码（但逻辑清晰，1 天内可 100% 覆盖并补齐单测）。
- **方案 B（备选）：保留 `fclover-core` (Rust) 编译为 `.dylib`，在 Go 中通过 `purego` 动态加载**
  - **最大获益点**：直接复用现有的 3 个 Rust 源码文件（约 900 行）。
  - **不可承受的代价 (Worst-case Cost)**：违背“消除双语言维护成本”的核心诉求；每次跨越 Go ↔ Rust 边界仍需 JSON 序列化/反序列化，且打包 `.app` 时必须维护 `@rpath/libfclover_core.dylib` 双二进制签名。
- **结论**：**坚定选择方案 A（100% 纯 Go）**。

### 决策 2：原生文件图标与图片缩略图管线
- **方案 A（主推）：`mygo.Shell.FileIcon` + Go 原生图片缩放 + 双层缓存（内存 LRU + 磁盘 SHA256 缓存）**
  - **最大获益点**：零 CGo 依赖；`.app`、文件夹、各类文档直接通过 `mygo.Shell.FileIcon` 获取 Finder 级原生图标；常见图片（PNG/JPEG/GIF/WebP）由后台 Worker 解码生成缩略图并持久化至 `~/.clover-go/cache/thumbs/`，二次进入目录 **0 毫秒直出**。
  - **不可承受的代价 (Worst-case Cost)**：对于视频（`.mp4`/`.mov`）和 `.pdf` 的封面缩略图，纯 Go 不内置解码器，需异步调用一次系统 `/usr/bin/qlmanage -t` 生成后落盘缓存。
- **方案 B（备选）：通过 `purego` 自行桥接 Objective-C `QLThumbnailGenerator`**
  - **最大获益点**：视频/PDF 缩略图无需拉起 `qlmanage` 子进程。
  - **不可承受的代价 (Worst-case Cost)**：`QLThumbnailGenerator` 强依赖 Objective-C Block 异步回调与 `CGImage` 转换，在 `CGO_ENABLED=0` 下手写 `purego` Block 签名容易引发引用计数泄漏。
- **结论**：**选择方案 A**（高频系统图标走 `mygo.Shell.FileIcon`，常规图片走纯 Go 极速缩略，冷门视频/PDF 按需异步调 `qlmanage` 并永久缓存至磁盘）。

### 决策 3：空格键 Quick Look 预览实现
- **方案 A（主推）：右侧常驻原生预览栏 (`⇧⌘P`) + 空格键通过 `purego` 轻量唤起系统 `QLPreviewPanel`（降级回退 `qlmanage -p`）**
  - **最大获益点**：代码、Markdown、图片、文件属性 90% 场景直接在右侧 Metal GPU 预览栏内 **0 延迟瞬时呈现**；按下 `Space` 时唤起 macOS 原生 Quick Look 面板，完美兼顾速度与系统级全格式预览能力。
  - **不可承受的代价 (Worst-case Cost)**：需编写约 80 行 ` darwin` 专属 `purego` 封装以管理 `QLPreviewPanel` 数据源。
- **方案 B（备选）：完全自研弹窗预览所有格式**
  - **最大获益点**：无任何系统 API 耦合。
  - **不可承受的代价 (Worst-case Cost)**：无法预览 PDF、Office、音视频与設計稿，严重削弱 Mac 文件管理器的核心竞争力。
- **结论**：**选择方案 A**。
