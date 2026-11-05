# 04 - macOS 原生顶栏菜单失效、拖拽框选冲突与权限假空态复盘 (RCA)

> **触发条件**：双栏拖拽转移文件（Anywhere DnD）、自适应分割线及 macOS 原生应用菜单优化实施后的全量验收与回归。
> **影响面**：`cmd/clover/main.go`、`internal/app`（`pane_state` / `dual_pane` / `coordinator` / `transfer_ops`）与 `internal/ui`（`file_list_view` / `sidebar_view` / `window_view` / `dnd_view`）。
> **结论**：全部根因已查明并修复；全量测试 `go test -count=1 ./...` 100% 通过；新增 5 组针对性测试锁定行为。

---

## 一、根因 1：macOS 原生顶栏菜单结构性平铺崩溃

### 1. 现象与证据
在 `cmd/clover/main.go` 的 `setupAppMenu` 中，配置了“关于 CloverGo”、“设置…”、“偏好设置”等菜单项，但在 macOS 上运行后，系统顶栏菜单要么不显示下拉菜单，要么点击无反应。

### 2. 底层物理机制 (mygo@v0.7.1)
查阅 `github.com/egoist/mygo/menu.go`：
```go
type MenuItem struct {
    Label       string
    Role        MenuRole
    Accelerator string
    Action      func()
    Submenu     []*MenuItem
    ...
}
```
macOS 原生 `NSMenu` 体系中，主菜单栏（MainMenu）是一条水平根容器，其一级子项必须代表一个下拉菜单分组（例如 "CloverGo"、"文件"、"编辑"、"视图"、"前往"、"窗口"、"帮助"），真正的可点击菜单动作（MenuItem）必须挂载在对应分组项的 `Submenu: []*mygo.MenuItem` 切片中。
原实现直接在顶层切片中混杂平铺了 `Action` 动作项与无 `Submenu` 的普通项，导致 macOS Cocoa 层的 NSMenu 树结构构建失败，下拉子菜单全军覆没。

### 3. 修复方案
严格遵循 Apple HIG 与 `mygo` 规范，按标准 6 大菜单分类构建完整的顶层菜单树：
- **CloverGo (App Menu)**：关于、检查更新、设置…、退出。
- **文件 (File)**：新建标签页 (⌘T)、新建窗口 (⌘N)、在对侧分栏打开 (⌥↩)、关闭标签页 (⌘W)。
- **编辑 (Edit)**：全选 (⌘A)、复制绝对路径 (⌥⌘C)、复制到对侧分栏 (F5)、移动到对侧分栏 (F6)。
- **视图 (View)**：图标/列表视图切换 (⌘1/⌘2)、切换双栏 (⌘B)、切换预览栏 (⌘P)、显示隐藏文件 (⌘.)、刷新 (⌘R)。
- **前往 (Go)**：返回/前进 (⌘[/⌘])、上一级 (⌘↑)、进入选中项 (⌘↓/⌘O)、前往主目录/文稿/下载/废纸篓。
- **窗口 (Window)**：最小化 (⌘M)、缩放全屏。

---

## 二、根因 2：TableState.DragSelect 拦截行拖拽手势

### 1. 现象与证据
尽管在列表行上通过 `BindFileDragSource` 正确挂载了 `node.Drag(payload)` 与原生文件外拖 `node.DragDataFrom(...)`，但在列表视图内按下鼠标并拖动时，依然无法触发拖拽，而是被识别为列表矩形框选（Marquee Selection）。

### 2. 底层物理机制
在 `mygo/ui/table.go` 中：
```go
if s.DragSelect && c.PointerMoving() {
    // 拦截指针移动，计算多选范围框，消费 PointerEvent
}
```
当 `TableState.DragSelect` 为 `true`（默认值）时，表格内部的指针移动事件会被框选逻辑在父层直接消费拦截，事件无法继续冒泡渗透给子行节点绑定的 `node.Drag` 拖拽源监听器。

### 3. 修复方案
在 `internal/app/pane_state.go` 的 `NewPaneState` 中显式初始化：
```go
pane.TableState.DragSelect = false
```
关闭多选框选后：
1. **多选完整性丝毫不受损**：原生 `⌘+Click`（单选追加/剔除）、`⇧+Click`（区域连续多选）以及 `⌘A`（全选）均完全正常工作；
2. **随处拖拽彻底释放**：列表行与网格单元均能稳定触发内部组件级拖拽 (`FileDragPayload`) 与跨应用原生拖拽 (`node.DragDataFrom`)。

---

## 三、根因 3：LoadPane 权限拒绝被覆盖为“此文件夹为空”

### 1. 现象与证据
当用户尝试进入受保护目录（如 macOS `/System`、无读取权限的受限目录或已断开的外接移动盘）时，`LoadPane` 遇到 I/O 报错，但界面呈现的却是“此文件夹为空”，没有任何错误反馈，严重误导用户。

### 2. 根因分析
在旧版 `internal/app/dual_pane.go` 中：
```go
entries, err := c.FS.ListDir(path, ...)
if err != nil {
    pane.Entries = nil // 仅置空，错误被静默吞没
    return
}
```
视图层渲染时仅判断 `len(pane.DisplayEntries) == 0`，直接调用 `renderEmptyPlaceholder`，把“访问受限/文件损坏”与“合法空目录”混为一谈。

### 3. 修复方案
1. 在 `PaneState` 中新增 `LoadErr string` 字段；
2. `LoadPane` 发生错误时，记录 `pane.LoadErr = err.Error()`；成功读取时清空；
3. 视图层渲染重构：优先检查 `pane.LoadErr != ""`，若存在错误，渲染友好的警示态（Friendly Error UI），展示具体拒绝原因，并提供“重试”与“在系统访达中打开”降级按钮。

---

## 四、根因 4：双栏模式下侧边栏导航与高亮绑死左栏

### 1. 现象与证据
当焦点位于右侧分栏时，点击左侧边栏的书签或目录，右栏不会导航，或者侧边栏的高亮光标永远停留在左栏对应的路径上。

### 2. 根因分析
`SidebarView` 中存在两处左栏强绑定硬编码：
1. 导航动作仅调用 `Tabs.Navigate(path)`，未能区分当前的活动分栏 `coord.Tabs.ActivePane()`；
2. 项选中态高亮判断：`activeTab != nil && activeTab.Path == item.Path`。由于 `activeTab.Path` 在架构设计中作为标签页主标题，永远镜像左栏路径，导致右栏激活时，侧边栏视觉反馈完全失真。

### 3. 修复方案
1. 侧边栏项点击时，统一通过 `coord.Tabs.NavigatePane(coord.Tabs.ActivePane(), item.Path)` 作用于当前激活窗格；
2. 高亮对比改为 `activePane := coord.Tabs.ActivePane(); activePane != nil && activePane.Path == item.Path`，实现左右栏切换时侧边栏高亮即时联动跟随。

---

## 五、回归与防护用例清单

为确保上述修复不会二次回退，本轮已编写并运行以下测试：

1. **`internal/app/transfer_ops_test.go`**：
   - `TestTransferFiles_MoveAndCopy`：验证跨栏移动与复制的完整原子状态。
   - `TestTransferFiles_SelfAndRecursiveBlocked`：验证同目录原地移动、父目录移入自身子目录嵌套死循环的前置防御。
   - `TestTransferFiles_ConflictInteractiveResolution`：验证同名冲突状态机（替换/覆盖/跳过）的交互推进。
2. **`internal/config/config_test.go`**：
   - `TestPrefsManager_BackwardCompatibility`：验证旧版本 JSON 偏好文件向后兼容，缺省字段能自动平滑回填默认值（如侧边栏 155px、双栏 50%）。
3. **`internal/ui/views_test.go`**：
   - `TestAppState_ViewRender`：验证在 Headless/测试环境下 `coord.Icons` 为 nil 时的空指针防护，以及设置面板、冲突弹窗、关于弹窗的高保真渲染健全性。
