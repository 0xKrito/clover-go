# 02 - 模块契约、代码目录规划与分阶段演进路线图 (Roadmap)

> **代码规模红线**：严格执行单文件 **≤400 行**、单函数 **≤50 行**，职责按领域垂直切分，严禁出现上帝模块（God Module）。

---

## 一、 项目目录结构规划 (`clover_go`)

```text
clover_go/
├── go.mod / go.sum                      # 模块名: github.com/bighaidao/clover-go (go 1.27.3, CGO_ENABLED=0)
├── mygo.json                            # macOS App Bundle 元数据与 fclover:// URL Scheme 配置
├── Makefile                             # 极简研发入口: make run / make test / make bundle / make dmg
├── cmd/
│   └── clover/
│       └── main.go                      # 唯一入口: 单实例锁 + 深链注册 + 主窗口启动 + 优雅退出
├── internal/
│   ├── config/                          # [Infra] 配置与持久化沙箱 (~/.clover-go/)
│   │   ├── paths.go                     #   沙箱路径收敛、单实例 Flock 文件锁
│   │   ├── prefs.go                     #   外观/主题色/默认视图/排序偏好 (Temp+Rename 原子写)
│   │   └── workspace.go                 #   收藏夹、顶部书签栏、命名工作区与会话快照持久化
│   ├── domain/                          # [Domain] 纯业务逻辑层 (100% 无 I/O，纯函数可测)
│   │   ├── file_item.go                 #   FileInfo 核心模型、人类可读大小/时间格式化、轻量类型归类
│   │   ├── sorter.go                    #   Finder 规则排序器 (修改日期混排 vs 名称/大小文件夹优先)
│   │   ├── batch_rename.go              #   批量重命名 4 种规则计算引擎与冲突预检
│   │   └── path_crumb.go                #   路径规范化、家目录前缀折叠、同名标签消歧副标题计算
│   ├── fs/                              # [Infra] 纯 Go 文件系统引擎 (替代原 Rust fs.rs)
│   │   ├── reader.go                    #   ReadDir 分页与流式读取、软链接解析、目录体积递归计算
│   │   ├── ops.go                       #   Copy / Move / CreateDir / Rename / Duplicate 及冲突策略执行
│   │   ├── archive.go                   #   调用系统 /usr/bin/ditto 实现 Zip 压缩与安全解压
│   │   └── volumes.go                   #   /Volumes 挂载卷探测与磁盘可用空间查询 (unix.Statfs)
│   ├── search/                          # [Infra] 内存缓存模糊搜索引擎 (替代原 Rust search.rs)
│   │   ├── index.go                     #   并发 WalkDir 索引构建与 30s TTL LRU 缓存
│   │   └── scorer.go                    #   多 Token 拆分、前缀/精确/子串匹配打分与排序
│   ├── macos/                           # [Infra] macOS 原生系统能力桥接 (CGO_ENABLED=0)
│   │   ├── colortag.go                  #   unix.Getxattr/Setxattr 读写 Finder 7 色标签 (bplist00 编解码)
│   │   ├── launcher.go                  #   打开终端 (Terminal/iTerm/Ghostty/Warp)、编辑器与「打开方式」
│   │   ├── quicklook.go                 #   空格键 Quick Look 预览触发与焦点跟随
│   │   └── finder_service.go            #   fclover:// 深链解析与 Finder 右键服务 .workflow 自动安装器
│   ├── icon/                            # [Infra] 原生文件图标与缩略图双层缓存管线
│   │   ├── pipeline.go                  #   LIFO 视口优先 Worker 池 (6 并发) + 内存 LRU + 磁盘缓存
│   │   └── thumb.go                     #   纯 Go 图片缩略图生成 + 视频/PDF 缩略图回退
│   ├── app/                             # [Application] 窗口与标签状态机编排层
│   │   ├── window_registry.go           #   多窗口管理、标签拖出新建窗口 (Detach) 与跨窗合并 (Merge)
│   │   ├── tab_manager.go               #   多标签增删改、Pin 固定、前进后退栈、LIFO 关闭恢复栈 (⇧⌘T)
│   │   └── coordinator.go               #   单窗口协调器: 目录加载代数控制(loadGen)、FSEvents 绑定、剪贴板与冲突调度
│   └── ui/                              # [Presentation] MyGo Metal GPU 立即模式视图层 (单文件 ≤350 行)
│       ├── async.go                     #   RunOnMain 主线程状态收敛封装
│       ├── style.go                     #   Apple HIG 双模调色板、9 色语义 Tint、毛玻璃材质适配
│       ├── icons.go                     #   全套 SF Symbols 风格纯矢量 *ui.SVG 图标库 (零 Emoji)
│       ├── widgets.go                   #   复用小组件: PillButton、ColorDot、BreadcrumbBar、EmptyCard
│       ├── window_view.go               #   单窗口根视图: 沉浸式红绿灯顶栏 + 侧边栏 + 主区 + 预览栏 + 状态栏
│       ├── tabbar_view.go               #   Clover 页感标签栏渲染、拖拽排序与拖出窗口手势对接
│       ├── toolbar_view.go              #   前进后退、路径面包屑/输入框切换、实时搜索框、书签栏
│       ├── sidebar_view.go              #   毛玻璃侧边栏 (收藏夹、命名工作区、磁盘卷、颜色标签过滤)
│       ├── file_list_view.go            #   Finder 式四列表格视图 (ui.Table + 列头排序 + 右键菜单)
│       ├── icon_grid_view.go            #   虚拟化图标网格视图 (ui.GridView + 尺寸缩放 + 雙击/选中)
│       ├── preview_pane_view.go         #   右侧常驻预览栏 (高清大图 / 代码高亮 / Markdown / 颜色标签点选)
│       └── modals_view.go               #   批量重命名弹窗、同名冲突解决弹窗、新建工作区与设置弹窗
├── scripts/
│   └── package-arm64.sh                 # 一键构建 CloverGo.app 与 CloverGo-arm64.dmg 脚本
└── docs/                                # 高内聚 MECE 文档体系 (本目录)
```

---

## 二、 核心模块接口契约 (Go Contracts)

### 2.1 文件系统与冲突策略契约 (`internal/fs`)
```go
type ConflictPolicy int

const (
    PolicyPrompt    ConflictPolicy = iota // 遇到冲突暂停并回调询问用户
    PolicyOverwrite                       // 覆盖目标
    PolicySkip                            // 跳过冲突项
    PolicyKeepBoth                        // 自动生成 "file 2.ext"
)

type ReadDirOptions struct {
    Path       string
    ShowHidden bool
    Offset     int
    Limit      int // 首屏传 200 实现瞬开，随后后台拉取全量
}
```

### 2.2 视口优先图标管线契约 (`internal/icon`)
- **渲染帧契约**：`icon.GetOrQueue(path string, mtime int64, size int64, px int, onReady func()) *ui.Bitmap`
  - 若内存 LRU 命中：立即返回 `*ui.Bitmap`（耗时 `<100ns`，零阻塞）；
  - 若未命中：返回内置矢量占位图标，并将请求压入 **有界 LIFO 栈（容量 64）**；后台 Worker 从栈顶取出当前用户正在看的条目，调用 `mygo.Shell.FileIcon` 或解码磁盘缓存，完成后经 `mygo.RunOnMain` 触发 `c.Services().Invalidate()` 刷新帧。

### 2.3 标签拖出成窗与合并契约 (`internal/app/window_registry.go`)
- 每个标签项挂载 `ui.Element.DragData(tabTransferData, transfer.DragOptions{ Done: onTabDragDone })`：
  - **窗口内/跨窗口标签栏释放**：目标窗口标签栏通过 `DropData` 接收 `TabTransfer`，插入对应位置并从源窗口移除（若源窗口标签数为 0 则调用 `sourceWin.Close()`）。
  - **拖出所有窗口在桌面空白处释放**：`onTabDragDone` 收到 `result.Canceled && result.Escaped` 且源窗口标签数 `>1` 时，读取鼠标屏幕绝对坐标 `result.Point`，直接调用 `Registry.SpawnDetachedWindow(tab, result.Point)` 在鼠标位置创建新窗口。

---

## 三、 分阶段实施路线图 (5 Stages Milestone)

```text
[Stage 1: 纯 Go 核心底座与领域单测]
  ├─ 初始化 go.mod (mygo v0.7.1) 与 Makefile / 打包脚本
  ├─ 实现 internal/domain (排序、格式化、面包屑折叠、批量重命名纯逻辑) + 单元测试
  └─ 实现 internal/fs (ReadDir 分页、Copy/Move 冲突策略、ditto Zip) 与 internal/search + 单元测试
         │
         ▼
[Stage 2: Apple HIG 视觉底座与 Clover 多标签单窗闭环]
  ├─ 落地 internal/ui/{style,icons,widgets,async}.go (毛玻璃顶栏、双模 9 色主题、全矢量 SVG)
  ├─ 落地 TabManager + TabBarView (新建/关闭/Pin固定/⇧⌘T恢复栈/同名副标题)
  └─ 落地 SidebarView + ToolbarView (路径面包屑、前进后退、书签栏)
         │
         ▼
[Stage 3: Finder 级双视图浏览与异步图标管线]
  ├─ 落地 IconPipeline (mygo.Shell.FileIcon + 图片缩略图 + 内存/磁盘双缓存)
  ├─ 落地 FileListView (四列排序、DragSelect 拖选、内联重命名、右键菜单)
  └─ 落地 IconGridView (ui.GridView 虚拟化网格、图标大小滑块、键盘二维导航)
         │
         ▼
[Stage 4: 预览栏、macOS 系统深水区联动与多窗口拆合]
  ├─ 落地 PreviewPaneView (图片/代码高亮/Markdown/属性/7色标签点选) + Space QuickLook
  ├─ 落地 Finder 7 色标签读写 (xattr)、终端/编辑器唤起、FSEvents 实时监听刷新
  └─ 落地 WindowRegistry (标签拖出成独立窗口 Detach / 拖回合并 Merge / fclover:// 深链)
         │
         ▼
[Stage 5: 全量自测、10 万文件压测与 .app/.dmg 交付]
  ├─ 运行 `go test -race ./...` 全绿 + 100k 文件目录流畅度与内存基准验证 (<45MB)
  └─ 执行 `./scripts/package-arm64.sh` 产出独立签名 `CloverGo.app` 与 `CloverGo.dmg`
```
