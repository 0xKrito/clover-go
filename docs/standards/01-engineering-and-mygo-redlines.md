# 01 - 编码规范、MyGo 渲染红线与质量验证准则 (Standards)

> 本文档凝练自全局架构规范与 `roxy/mygo-app` 实战踩坑教训，作为 `CloverGo` 编码与 Code Review 的强制基线。

---

## 一、 MyGo Metal GPU 立即模式四大铁律

### 1. 渲染帧 (`View(c *ui.Context)`) 绝对零 I/O 原则
- `View(c *ui.Context)` 每秒被 Metal 渲染管线调用 60~120 次。
- **严禁在 `View` 及其调用的子视图函数中执行**：
  - 任何文件/目录系统调用（`os.Stat`、`os.ReadDir`、`filepath.EvalSymlinks`、`unix.Getxattr`）；
  - 任何原生桥接图标提取（`mygo.Shell.FileIcon`）或图片解码（`image.Decode`）；
  - 任何 SQLite / JSON 磁盘读写或外部子进程调用（`exec.Command`）。
- **正确做法**：视图层只允许做 $O(1)$ 的内存 Map/Slice 读取；未命中的数据统一投递至后台 Worker，完成后调用 `mygo.RunOnMain` 更新状态并触发重绘。

### 2. 主线程状态变更纪律 (`mygo.RunOnMain`)
- `WindowSession`、`TabState` 等 UI 状态结构体**仅允许在 UI 主线程修改**。
- 后台 Goroutine 完成目录读取、搜索、复制或压缩后，必须将结果打包通过 `mygo.RunOnMain(func() { ... })` 切回主线程，校验 `loadGen` 后再写入状态切片。

### 3. 立即模式 API 规范与组件 Key 唯一性
- 弹性布局填充统一使用 `.Grow(1)`（而非 Web 习惯的 `.Flex(1)`）；
- 表格与网格行内控件必须绑定稳定的业务主键（如文件绝对路径 `item.Path` 或 `tab.ID`），严禁在动态过滤列表中使用纯数组下标作为状态 Key，防止帧间焦点串位。

### 4. 快捷键焦点防误触
- 全局或窗口级单键快捷键（如 `Space` 触发 Quick Look、`Return` 触发重命名、`⌫` 触发删除）在触发前，**必须检查当前是否有文本输入框（搜索框、路径编辑框、内联重命名框）处于聚焦或 IME 拼音组合输入状态**；若有，优先交由输入框处理。

---

## 二、 代码结构与工程环境规范

1. **纯 Go 零 CGo**：全程 `CGO_ENABLED=0`，Go 版本锁定 `go 1.27.3`（复用系统 `~/go` 标准 GOPATH，禁止在项目目录内生成 `.agy_home/go` 污染仓库）。
2. **文件与函数体量红线**：
   - 单个 `.go` 源文件严格 **≤ 400 行**；
   - 单个函数严格 **≤ 50 行**；
   - 严禁将多个领域的视图堆砌在单个 `app.go` 中。
3. **禁止静默失败 (No Silent Failure)**：
   - 任何文件复制、移动、删除、重命名、压缩、深链打开若返回 `error`，必须通过 `Toast` 或内联错误态向用户展示可读的中文原因。

---

## 三、 质量验证门禁

1. **单元测试与竞态检测**：每个阶段交付前必须实际运行 `go test -race ./...` 并确保 100% 通过。
2. **大目录性能门禁**：内置 100,000 文件目录生成与分页读取/排序基准测试（`BenchmarkReadAndSort100k`），确保首屏加载 `< 20ms`，UI 帧率稳定不卡死。
3. **打包产物门禁**：通过 `./scripts/package-arm64.sh` 产出独立 `CloverGo.app` 与 `.dmg`，验证无源码树路径依赖、双击即用、内存占用 `< 50MB`。
