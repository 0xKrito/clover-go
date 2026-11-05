# CloverGo 文档总索引 (Documentation Index)

> 本知识库严格遵循 **MECE 原则（一事一地，零冲突）** 构建，覆盖 `CloverGo`（基于 `github.com/egoist/mygo` v0.7.4 的纯 Go 原生 macOS 多标签文件管理器）的产品规格、UI/UX 原型、系统架构与工程规范。

---

## 🌐 可视化评审门户（浏览器一键打开）

- 🖥️ **[CloverGo 全景架构与产品原型可视化评审网页 (`docs/review-portal.html`)](review-portal.html)**  
  单文件内联 SVG 架构拓扑图、高保真 Mac 原生界面交互预览、模块拆解与五阶段路线图，支持直接在浏览器打开评审。

---

## 📋 事前：需求对齐与产品原型 (`docs/requirements/`)

- **[01 - CloverGo 产品需求与业务状态矩阵规格书 (PRD)](requirements/01-product-requirements-and-state-matrix.md)**  
  涵盖需求四维漏洞体检、6 大核心产品能力域拆解（Clover 多标签与跨窗拆合、Finder 双模浏览、可信文件操作与冲突状态机、双轨预览、macOS 系统深水区联动、Apple HIG 主题与工作区）及全量业务状态流转表。
- **[02 - MyGo 原生 GPU 界面交互与高保真原型设计 (`package ui`)](requirements/02-ui-ux-and-mygo-native-wireframes.md)**  
  涵盖列表视图 + 右侧常驻预览栏、图标网格视图、批量重命名与同名冲突弹窗的 Unicode 高保真原型线框图、四种极限交互态定义（空态/60ms 防抖加载态/报错恢复态/10万文件极端态）与 `mygo/ui` 原生组件映射表。
- **[03 - CloverGo 与原版 Clover (fclover) 深度差距分析与架构溯源报告](requirements/03-clover-product-gap-analysis-and-alignment.md)**  
  基于生产运行截图全面对齐：侧边栏「常用位置」与「收藏」双分组架构与右键收藏、地址栏一体化胶囊与可点击面包屑、设置齿轮入口、空格键 Quick Look 预览与快捷键死代码溯源、MyGo 顶部窗口拖拽 Bug 根因与 1:1 高保真 Unicode 原型对齐。
- **[04 - DirDeck 竞品深度对标与 CloverGo 关键特性实施路线图 (Benchmark & Roadmap)](requirements/04-dirdeck-benchmark-and-feature-roadmap.md)**  
  对标 DirDeck 瑞士军刀级工作台：深度拆解 14 组高保真功能截图，明确采纳「左右对等双栏、横向多层级书签栏、批量重命名可视化规则链、压缩包就地免解压穿透虚拟浏览及 8 大核心高频右键联动」，并依据奥卡姆剃刀定律剔除冗余悬浮球组件。
- **[05 - CloverGo 后续演进功能全栈 Unicode 高保真交互原型设计 (UI/UX Spec)](requirements/05-next-phases-unicode-prototypes.md)**  
  涵盖阶段二压缩包虚拟浏览（Archive VFS）、PDF 瑞士军刀微工具、图片文字水印处理、从 Finder 导入书签及工作区快照管理的 Unicode 高保真线框图与四种极限交互态定义。
- **[06 - 全场景随处拖拽 (Anywhere DnD) 与自适应分栏产品规格与交互原型 (UX Spec)](requirements/06-anywhere-drag-drop-and-dual-pane-ux-spec.md)**  
  基于 macOS 原生 Finder 与 ForkLift 体验对标：全场景拖拽（双栏互拖、子目录移入、面包屑上移、Tab栏转移、侧边栏收藏、原生文件拖入拖出）、自适应可调整分割线（侧边栏、双栏中缝、右侧预览栏）、原生应用菜单重构及同名冲突决策弹窗的高保真原型与状态矩阵。

---

## 🏛 事中：系统架构与模块契约 (`docs/architecture/`)

- **[01 - CloverGo 全栈纯 Go 系统架构与拓扑设计 (RFC)](architecture/01-system-architecture-and-topology.md)**  
  涵盖三类标准 Unicode 拓扑图（单向依赖分层图、Authoritative Store vs Derived State 数据流向图、资源生命周期与确定性销毁 Teardown 图）、架构五维审查清单与三大核心技术方案对比推演。
- **[02 - 模块契约、代码目录规划与分阶段演进路线图 (Roadmap)](architecture/02-module-contracts-and-roadmap.md)**  
  涵盖 `clover_go/` 全量目录树规划（严格遵守单文件 ≤400 行、单函数 ≤50 行）、文件系统/LIFO 图标管线/跨窗标签拆合核心 Go 接口契约，以及 5 阶段可执行落地里程碑。
- **[03 - 全场景拖拽 (Anywhere DnD)、自适应分割线与应用菜单架构设计 (RFC)](architecture/03-anywhere-dnd-and-adaptive-splitters-architecture.md)**  
  涵盖 MyGo 原生双轨拖拽载荷生命周期、自适应布局三分割线架构拓扑、交互式文件冲突解决状态机、macOS 原生 NSMenu 分层映射契约与单向门/双向门演进决策。

---

## 📏 基线：工程规范与质量红线 (`docs/standards/`)

- **[01 - 编码规范、MyGo 渲染红线与质量验证准则 (Standards)](standards/01-engineering-and-mygo-redlines.md)**  
  凝练自 `roxy/mygo-app` 实战经验的四大铁律（120Hz 渲染帧零 I/O、`RunOnMain` 主线程纪律、组件唯一业务 Key、快捷键焦点防误触）及测试/打包验收标准。

---

## 🔧 事后：排错记录与根因分析 (`docs/troubleshooting/`)

- **[01 - 首版实现全量代码与体验深度审计报告 (Architecture, Product & UI/UX Review RCA)](troubleshooting/01-stage1-5-implementation-review-rca.md)**  
  针对 commit `874a301`~`c0588f0` 及运行截图的深度取证审计：剖析“顶部被劈成三截”的 `mygo/ui` Flexbox 交叉轴根因、9 项“仅写单测未接 UI”的假交付功能、`.app` 识别与图标管线 Bug，以及右键菜单/快捷键缺失清单。
- **[02 - 阶段一双栏 P0「操作绑死左栏」与批量重命名失真复盘 (RCA)](troubleshooting/02-dual-pane-left-binding-and-batch-rename-rca.md)**  
  PaneState 拆分后操作层未跟随 pane 化导致右栏操作全部落到左栏（含重命名改错文件的数据完整性风险）的根因与修复；批量重命名预览/执行参数漂移、错误静默与磁盘同名覆盖拦截；含 6 个回归测试与残留风险清单。
- **[03 - 阶段二 5 特性 Review 修复复盘：Zip Slip、水印格式损坏与 Swift 转义 (RCA)](troubleshooting/03-stage2-five-features-review-rca.md)**  
  压缩包虚拟浏览的 Zip Slip 目录逃逸（P0 安全）、水印 keep 格式对 HEIC/WEBG/GIF 输出的损坏、tar/tar.gz 提取链路缺口、Go %q 拼接 Swift 源码的转义不兼容、固定名覆盖与临时文件生命周期，及 pane 纪律延续检查结论；含 4 组新增回归测试。
- **[04 - macOS 原生顶栏菜单失效、拖拽框选冲突与权限假空态复盘 (RCA)](troubleshooting/04-macos-app-menu-and-table-dragselect-rca.md)**  
  深入排查修复：macOS 顶层菜单树扁平化导致的 NSMenu 失效、`TableState.DragSelect` 拦截行拖拽手势根因、`LoadPane` 权限拒绝被覆盖为“此文件夹为空”的假空态缺陷、双栏模式下侧边栏点击与高亮绑死左栏，以及 headless 渲染下的空指针防御。
- **[05 - 列表各维度列宽拖拽缺失与默认排序未生效复盘报告 (RCA)](troubleshooting/05-column-resize-and-default-sort-rca.md)**  
  列表表头各列（名称、创建时间、修改时间、大小、类型）交互式拖拽手柄、双击恢复默认、超宽横向滚动与像素级表头表体对齐实现；启动时第 0 标签页默认排序时序倒置根因修复及偏好设置修改时全局窗格即时重排落地。
