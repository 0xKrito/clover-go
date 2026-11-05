# 架构方案：全场景随处拖拽 (Anywhere DnD)、自适应分栏与原生 macOS 菜单体系 (RFC)

> 本文档规范 CloverGo 全场景随处拖拽引擎、可调分栏分割线系统（侧边栏/双栏/预览栏）及 macOS 顶层应用级菜单树的架构设计与契约约束。

---

## 一、 系统架构直观可视化（标准 Unicode 拓扑）

### 1.1 分层依赖与边界拓扑（单向无环依赖）

```text
┌────────────────────────────────────────────────────────────────────────────┐
│  Host & Native Platform Layer (cmd/clover/main.go)                         │
│  [ setupAppMenu ] ──> macOS 原生顶层 Submenu 树 (CloverGo/文件/编辑/视图/前往)│
└─────────────────────────────────────┬──────────────────────────────────────┘
                                      │ 单向调用
                                      ▼
┌────────────────────────────────────────────────────────────────────────────┐
│  Presentation Layer (internal/ui/ — 120Hz 声明式视网膜渲染，零阻塞 I/O)     │
│  ├─ [ window_view.go ]      : ui.Split(侧栏) & SplitPercent(双栏) 拖拽调节 │
│  ├─ [ dnd_view.go ]         : BindFileDragSource / BindFileDropTarget      │
│  ├─ [ settings_modal.go ]   : 分栏式设置中心 + 关于 CloverGo 对话框        │
│  ├─ [ conflict_modal.go ]   : 同名冲突解决弹窗 (替换/保留两者/跳过/应用全部)│
│  ├─ [ preview_pane_view.go ]: PreviewWidth 自适应左侧拉手与元数据          │
│  └─ [ file_list_view.go / icon_grid_view.go / sidebar_view.go / tabbar ]   │
└─────────────────────────────────────┬──────────────────────────────────────┘
                                      │ 意图派发 (Intent Dispatch)
                                      ▼
┌────────────────────────────────────────────────────────────────────────────┐
│  Application & Domain Layer (internal/app/ — 业务编排与并发控制)           │
│  ├─ [ transfer_ops.go ]     : FileDragPayload, TransferFiles, 冲突状态机   │
│  ├─ [ dual_pane.go ]        : PaneState.LoadErr 隔离与双栏对侧互转操作     │
│  ├─ [ pane_state.go ]       : TableState.DragSelect 释放与行拖拽手势接管   │
│  └─ [ coordinator.go ]      : 持运行时布局尺寸镜像与 Toast 通知机制        │
└─────────────────────────────────────┬──────────────────────────────────────┘
                                      │ 适配器调用
                                      ▼
┌────────────────────────────────────────────────────────────────────────────┐
│  Infrastructure Layer (internal/fs/ & internal/config/)                    │
│  ├─ [ fs/ops.go ]           : ValidateTransfer(防父进子死循环/防自覆/防VFS)│
│  └─ [ config/prefs.go ]     : SidebarWidth, DualPaneRatio, PreviewWidth,   │
│                               DragAction, ConflictAction 强类型持久化      │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## 二、 数据流向与权威真理源图 (Store vs Derived State)

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ Authoritative Store (权威真理源)                                           │
│  1. 磁盘持久化配置: PrefsManager (`prefs.json`)                            │
│     - SidebarWidth (默认 155), DualPaneRatio (默认 50), PreviewWidth (250) │
│     - DragAction ("move"/"copy"), ConflictAction ("ask"/"rename"/...)      │
│  2. 真实文件系统: OS FileSystem + FSEvents DirWatcher                     │
│  3. 挂起事务状态: Coordinator.PendingConflict (待决同名冲突)               │
└──────────────────┬───────────────────────────────────────┬─────────────────┘
                   │ 启动解析 / 防抖回写                   │ 异步读取 / 变更重刷
                   ▼                                       ▼
┌────────────────────────────────────────┐ ┌─────────────────────────────────┐
│ Derived Runtime Layout State (UI 镜像) │ │ Derived Pane Snapshot (只读镜像)│
│  • Coord.SidebarWidth (float32: 155)   │ │  • PaneState.Entries            │
│  • Coord.DualPaneRatio (float32: 50)   │ │  • PaneState.DisplayEntries     │
│  • Coord.PreviewWidth (float32: 250)   │ │  • PaneState.LoadErr (错误区分) │
│  (ui.Split 读写该指针，仅在 Changed 时 │ │  (UI 零修改权限，仅通过         │
│   持久化回写，彻底杜绝高频磁盘 I/O)    │ │   LoadPane 在主线程置换更新)    │
└────────────────────────────────────────┘ └─────────────────────────────────┘
```

---

## 三、 状态机与资源生命周期 (Lifecycle & Teardown)

```text
[拖拽手势发起] ──(拖拽移动 > 4 DIPs)──> 打包内存态 FileDragPayload{Paths}
       │
       ├──(按 Esc 或拖至窗口外释放)──> mygo 自动重置 runtime.drag ──> 零副作用结束
       │
       └──(在合法 DropTarget 释放)──> 纯函数校验 fs.ValidateTransfer(srcs, dstDir)
                                           │
               ┌──(校验不通过: 同目录/父进子死循环/只读VFS)──> Coordinator.Toast(err)
               │
               └──(校验通过)──> 检查目标是否存在同名文件
                                   │
                                   ├─ 无冲突 ──> 后台协程执行 MoveItem / CopyItem
                                   │
                                   └─ 有冲突 ──> 挂起为 PendingConflict
                                                 弹出 ConflictModal
                                                 (Esc / 取消按钮直接释放销毁)
                                                 (选择后按策略推进剩余文件)
                                   ▼
                       执行完成 ──> runOnMain 刷新双栏与对应标签页
```

---

## 四、 架构五维审查清单自检

1. **状态归属**：
   - 布局尺寸真理源在 `Preferences`；运行时由 `Coordinator` 维护 `SidebarWidth`、`DualPaneRatio`、`PreviewWidth`，绑定 `ui.Split`，仅在拖拽停顿/变更时写入磁盘，保障 120Hz 丝滑拖拽。
2. **生命周期**：
   - 冲突弹窗采用非阻塞事件队列推进，不使用阻塞 Goroutine 通道，Esc/关闭窗口即时释放，无资源泄漏风险。
3. **异步时序**：
   - 文件转移在后台 Goroutine 异步进行，完成后通过 `runOnMain` 回调通知，并伴随代次号检查，防止旧异步结果覆盖当前导航路径。
4. **爆炸半径**：
   - `fs.ValidateTransfer` 在 I/O 执行前拦截：父目录拖入自身子目录（循环嵌套死锁）、拖入自身（No-op）、拖入只读压缩包 VFS，全面杜绝数据损毁。
5. **单双向门决策**：
   - `prefs.json` 架构字段扩展为单向门升级：通过 `NewPrefsManager` 内置边界保护，老配置文件缺省字段自动回填推荐默认值，100% 向后兼容。
