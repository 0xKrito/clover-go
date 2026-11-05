# 列表各维度列宽拖拽缺失与默认排序未生效复盘报告 (RCA)

> **问题分类**：交互细节与初始化时序缺陷  
> **涉及模块**：`internal/ui/file_list_view.go`, `internal/ui/settings_modal.go`, `internal/app/coordinator.go`, `internal/config/prefs.go`  
> **影响范围**：长文件名展示受阻、偏好设置中“按时间倒排”在首屏和切换时无法生效

---

## 一、 问题现象与根因取证

### 1. 维度拖拽功能缺失 (列宽无法调整)
- **现象**：文件列表表头各列（名称、创建时间、修改时间、大小、类型）之间没有拖拽手柄。当文件名很长时，由于「名称」列被固定或挤压，文件名被省略截断，无法通过拖拽拉宽来查看完整名称。
- **根因**：
  - `renderTableHeaders` 与 `renderMetaCells` 仅使用硬编码固定宽度（110px、110px、70px、100px）和静态间距 `Gap(12)`；
  - 表头各列之间未放置交互式分割手柄（Resize Handle），缺乏 `ui.CursorResizeEW` 鼠标光标与 `Dragged()` 手势监听；
  - 缺乏列宽在 `config.Preferences` 和 `Coordinator` 中的状态维护与持久化机制。

### 2. 默认排序设置未生效
- **现象**：在偏好设置中将默认排序设定为“修改时间/创建时间 + 最新在前”，但重新打开或当前列表依然显示为按“名称 ▲”（正序）排序。
- **根因**：
  - **启动初始化时序倒置**：`NewCoordinator` 优先调用了 `NewTabManager(initialPath)`，在构造首个 Tab 0 时 `TabManager.DefaultSort` 尚为初始默认值 `name/asc`。随后从配置读取的默认排序虽赋给了 `c.Tabs.DefaultSort`，却未同步至已生成的 `Tabs[0].PaneState.Sort`，导致应用启动后的第一个标签页永远默认按名称升序。
  - **偏好变更只改模板、未刷当前视图**：`applyDefaultSort` 仅改写了 `coord.Tabs.DefaultSort`（注释标记“当前窗格不受影响”），未驱动当前所有已打开的 Tab 和左右分栏执行 `domain.SortFiles` 与 `ApplyPaneFilter`，导致用户即使在设置弹窗中修改了排序，也看不到任何界面响应。

---

## 二、 修复方案与工程落地

### 1. 列表表头与表体对齐的交互式列宽拖拽 (`internal/ui/file_list_view.go`)
- 引入 `renderColDivider(c, onDrag, onReset)`：
  - 8px 点击响应区，中置 1px 高对比度分割线（悬停高亮为 `t.Accent`）；
  - 绑定 `ui.CursorResizeEW` 光标，拖拽实时累加位移 `dx`；
  - 双击手柄触发重置为标准默认宽度。
- 表头与列表行精确对齐：
  - 移除硬编码 `Gap(12)`，以显式 8px 单元格间距与列宽（`ColWidthName`, `ColWidthCreateTime`, `ColWidthModTime`, `ColWidthSize`, `ColWidthKind`）严格匹配；
  - 结合 MyGo 原生 `RowWidth(totalW)` 与 `head.ScrollWith(list)`，当拉伸列宽超出可视区域时自动启用平滑横向滚动，表头与表体始终保持像素级对齐。

### 2. 默认排序启动绑定与即时响应 (`internal/app/coordinator.go` & `internal/ui/settings_modal.go`)
- **启动顺序修复**：在 `NewCoordinator` 构造末尾将 `sortOptionsFromPrefs(p)` 同步覆盖写入 `c.Tabs.Tabs[0].PaneState.Sort` 及可能存在的右栏。
- **即时重排联动**：新增 `SetDefaultSortAndResort(field, dir)` 方法：
  - 更新 `c.Tabs.DefaultSort`；
  - 遍历所有现有 Tab 及其左右两栏，重新赋值 `Sort`，立即调用 `domain.SortFiles` 与 `ApplyPaneFilter` 并触发 `c.Invalidate()`。

### 3. 列宽偏好持久化与重置 (`internal/config/prefs.go`)
- 在 `Preferences` 增加 `ColWidthName`, `ColWidthCreateTime`, `ColWidthModTime`, `ColWidthSize`, `ColWidthKind` 字段及安全极值钳制（如名称 80~1200px）。
- 在偏好设置「拖拽与分栏」中将重置按钮升级为“重置分栏比例与各列宽度”，一键恢复出厂布局。

---

## 三、 回归测试验证

- `internal/config/config_test.go`: 验证列宽字段缺省初始化与旧版本配置的向下兼容；
- `internal/app/coordinator_test.go`:
  - `TestCoordinator_DefaultSortAppliedOnStartup`: 验证首个标签页完全继承偏好设置（如 `create_time/desc`）；
  - `TestCoordinator_SetDefaultSortAndResort`: 验证修改默认排序时实时对现有条目重排；
  - `TestCoordinator_ColWidthsUpdateAndReset`: 验证列宽拖拉累加、上下限约束与双击重置；
- `internal/ui/views_test.go`:
  - `TestFileListView_AdaptiveWindowAndColumnResize`: 验证窗口在 600px、1000px、1400px 自适应缩放及列宽微调无漂移。

---

## 四、 增量优化：自适应弹性伸缩与极简时间格式

### 1. 窗口缩放自适应对齐（告别居中漂移与歪扭）
- **现象**：窗口拉宽或缩小时，表体数据行相较于表头出现左右浮动、脱节对齐。
- **根因**：`mygo/ui/base.go` 中 `coreButtonBase` 默认启用了 `.Center()`，导致宽屏下数据行向中轴聚集。
- **修复**：
  - 行容器交互层显式声明 `.Justify(ui.Start)`，锁定绝对左对齐基线；
  - 表头与表体「名称」列统一设为 `Grow(1)` 弹性自适应列，窗口拉宽时长文件名自动展开占满剩余空间，窗口缩小时优雅截断；
  - 表头按钮与单元格文字均设为 `Padding(2, 0)`，X 轴坐标 100% 垂直镜像贯通。

### 2. 时间格式规范化 (`2026/9/20 14:40`)
- **现象**：原有动态算法出现“某月某日”不带年份、“破折号连接”与“今天/昨天”混杂。
- **修复**：在 `internal/domain/file_item.go` 中统一定长为 `2006/1/2 15:04`（如 `2026/9/20 14:40`），年份永远显式可查、去除冗余破折号、紧凑优雅，各行高度整齐划一。

---

## 五、 进阶对齐：Finder 原生内联重命名、列显示开关与 MyGo v0.7.4 深度融合

### 1. 内联重命名视觉与交互对齐 Finder (`internal/ui/widgets.go`, `internal/ui/file_list_view.go`, `internal/app/file_ops.go`)
- **视觉反差与双层蓝底修复**：
  - `ui.List` 对 `Chosen` 行原生绘制 `Background(t.Accent)` 并把外层默认字色置为 `t.AccentText (#FFFFFF)`；
  - 选中态不再二次叠加 `row.Background(t.Selection)`，且在非重命名选中行（`invertText := isChosen && !isRenaming`）将文件名、图标、元数据文字统一反白（`t.AccentText` / `t.AccentText.Alpha(0.92)`）；
  - 进入内联重命名（`isRenaming`）时，行底显式切换为 `t.Surface` 遮蔽外层深蓝底，输入框显式指定 `.Background(t.Background).Border(1.5, t.Accent).TextColor(t.Text).CaretWidth(1.5)`，彻底消除“深蓝底配刺眼白框、白底配白字看不清”的缺陷。
- **Finder 原生交互闭环**：
  - **回车触发**：打通 `list.Submitted()`、`grid.Submitted()`、`hit.Shortcut(0, ui.KeyEnter)` 与全局 `handleItemShortcuts` 的 `KeyEnter`；
  - **自动预选文件名主干**：新增 `FilenameStemRunes(name, isDir)`，結合 `mygo v0.7.4` 的 `input.SetTextSelection(0, stemEnd)`，重命名文件时自动跳过 `.ext` 扩展名（兼容 `.gitignore` 等隐藏点文件与目录全选）；
  - **Esc 取消与失焦自动保存**：绑定 `input.OnTextKey` 拦截 `ui.KeyEscape` 立即取消，跟踪 `RenameFocusedOnce` 在失焦或点击其他行/空白区时自动 `CommitRename`；
  - **新建文件夹联动**：`CreateNewFolder` 落盘刷新后自动定位新目录并进入内联重命名。

### 2. 表头第一根分割线反向 Bug 修复与右键列显隐菜单 (`internal/ui/file_list_view.go`, `internal/config/prefs.go`)
- **第一根分割线正向拖拽**：移除原先将「名称｜创建时间」分割线绑定到 `UpdateColWidth("create_time", -dx)` 的反向逻辑；默认「名称」列 `Grow(1)` 弹性占满剩余宽度，用户拖拽第一根分割线时激活 `NameColManual = true` 并正向更新 `ColWidthName`，双击分割线恢复弹性自适应。
- **紧凑默认列宽与表头右键菜单**：默认保留全部 5 列，将元数据列收紧为 `创建时间 125px / 修改时间 125px / 大小 76px / 类型 88px` 并启用 `.FontFeatures("tnum")` 等宽数字对齐；表头绑定 `ContextMenu` 支持勾选切换「创建时间 / 修改时间 / 大小 / 类型」显隐及「重置所有列宽」，状态持久化至 `Preferences`。

### 3. MyGo v0.7.4 新特性全面融入
- `ui.SearchField`：替换工具栏手写搜索框，原生自带放大镜、一键清除 `×` 与 `Esc` 清空；
- `ListState.Label` & `GridState.Label`：激活列表与网格视图原生键盘敲字母快速定位文件（Type-to-Select）；
- `Element.Scrollbars(false)` & `Element.IgnoreInput(true)`：隐藏面包屑横向滚动条，让文件图标与标签圆点完全穿透鼠标事件；
- `c.Toast`：将状态栏提示同步桥接至 `mygo` 原生浮动 Toast。
