# 02 - 阶段一双栏 P0「操作绑死左栏」与批量重命名失真复盘 (RCA)

> **触发条件**：阶段一（左右对等双栏 + 书签栏 + 8 大右键联动 + 批量重命名预览）交付后的全量 Review。
> **影响面**：`internal/app`（coordinator / file_ops / tab_manager / pane_state）与 `internal/ui`（8 个视图文件）。
> **结论**：全部修复并通过 `go test ./...` + `go vet` + `go build`；新增 6 个回归测试锁定行为。

---

## 一、根因：PaneState 拆分只拆了"读"，没拆"写"

阶段一将目录浏览状态抽象为 `PaneState`（`internal/app/pane_state.go`），`TabState` 通过 Go 结构体嵌入保留左栏并新增 `RightPane`。但全部**条目级操作方法**仍只读嵌入的左栏字段：

```text
【修复前】双栏模式下的操作路由（错误）
                 ┌─ tab.Chosen / tab.DisplayEntries / tab.Path ◄── 永远是左栏
Coordinator ────┤
                 └─ tab.RightPane（仅渲染使用）

【修复后】
                 ┌─ pane.Chosen / pane.DisplayEntries / pane.Path
Coordinator ────┤   （UI 传入被交互的 pane，快捷键传 ActivePane()）
                 └─ tab 级字段仅承载标签页外观（Path/Title 跟随左栏）
```

**后果清单（修复前）**：
1. **重命名改错文件**：右栏右键"重命名" → `StartRename(i)` 用左栏 `DisplayEntries` 定位，提交时静默改掉左栏同序号文件（数据完整性事故级）。
2. 剪切/复制/粘贴/废纸篓/创建副本/压缩/解压/批量重命名全部取左栏选中集；粘贴与新建文件夹落点为左栏目录。
3. 右栏双击文件夹在左栏打开（`openItem` 走 `Tabs.Navigate`=激活窗格）。
4. 空格 QuickLook / Enter 重命名作用于左栏残留选中。
5. 地址胶囊与搜索框显示左栏路径，导航却作用于激活窗格（显示与行为不一致）。

**修复方案**：所有操作方法以 `*PaneState` 为参数（`internal/app/file_ops.go`），新增 `TabManager.NavigatePane`（`internal/app/tab_manager.go`）——右栏导航不回写 `tab.Path/Title`；内联重命名新增 `RenamingPane` 指针防跨栏提交。

## 二、批量重命名：预览与执行"两张皮"

1. **参数漂移**：预览用 `Padding:3 / Separator:"-"`，执行用 `Padding:2 / Separator:"_"` → 序号模式预览 `photo-001`、实际产出 `photo_01`。
   **修复**：收敛为单一 `Coordinator.BatchRenameOptions()`（`internal/app/file_ops.go`），预览与执行共用。
2. **错误静默**：预览 `pairs, _ :=` 吞掉校验错误；执行错误被 `_ =` 丢弃，冲突时弹窗无任何反馈。
   **修复**：预览表内联渲染错误（红字）；执行错误写入 `BatchRenameError` 并在弹窗内展示，失败保持弹窗打开。
3. **磁盘覆盖风险**：`CalculateBatchRename` 只查批内重复，目标名撞上**未选中**的既有文件时 `os.Rename` 直接覆盖（数据丢失）。
   **修复**：执行前 `checkBatchRenameCollisions` 逐个 `os.Stat` 拦截；大小写仅差异的重命名豁免（macOS 文件系统大小写不敏感）。批内互换式目标同样被拦截（顺序执行会互相覆盖）。

## 三、顺带修复与卫生

- `Chosen` 从不随导航清空 → `PaneState.Navigate/GoBack/GoForward` 现在清空选择（Finder 语义），状态栏不再残留旧选中计数。
- "刷新 (⌘R)"提示存在但快捷键未注册 → `handleShortcuts` 补注册 ⌘R。
- 死代码清理：`SortByColumn`、`FileListView`/`IconGridView` 包装版（无调用方）。注意 `ToggleView` 被 `cmd/clover/main.go` 菜单使用，**不是**死代码（本轮曾误删后恢复，教训：死代码判定必须全仓 grep，不能限定 `internal/`）。
- 红线收敛：`coordinator.go` 由 712 行拆为 `coordinator.go`(344) + `file_ops.go`(361) + `preview.go`(83)；`handleShortcuts`(~120 行) 拆为 4 个职责函数；`ShowFileContextMenu`(~134 行) 拆为 5 个分区函数。剩余 58~74 行的存量 UI 渲染函数留待后续专门治理。

## 四、已知残留（未修复，需后续决策）

1. **书签多层级"能显示不能造"**：`AddBookmark` 只平铺追加，无创建子层级 UI；含子项的父书签不可点击直达、无右键移除。
2. **watcher 只跟随激活窗格**：非激活窗格的外部变更不自动刷新。
3. **跨窗格拖拽复制/移动**：规格书 04 承诺但未实现（阶段一范围争议项）。
4. 需真机手测：窗格点击激活的事件冒泡、书签 `btn.Menu` 下拉行为。

## 五、回归测试（`internal/app/file_ops_test.go` 等）

- `TestSelectedItems_PaneIsolation`：右栏选中对左栏不可见。
- `TestNavigatePane_TabTitleIsolation`：右栏导航不污染标签页标题。
- `TestPaneNavigate_ClearsSelection`：导航清空选择。
- `TestCheckBatchRenameCollisions`：占用目标/大小写豁免/未变更跳过。
- `TestExecuteBatchRename_CollisionBlocked`：冲突拒绝执行且错误入弹窗。
- `TestBatchRenameOptions_PreviewExecutionParity`：预览对即执行所得。
