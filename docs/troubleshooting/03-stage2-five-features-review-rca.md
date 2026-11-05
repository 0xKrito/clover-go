# 03 - 阶段二 5 特性 Review 修复复盘：Zip Slip、水印格式损坏与 Swift 转义 (RCA)

> **触发条件**：阶段二（Archive VFS / PDF 微工具 / 图片水印 / Finder 导入 / 工作区快照）交付后的全量 Review。
> **影响面**：`internal/fs`（archive_vfs/archive_extract）、`internal/macos`（pdf/watermark/finder_sidebar/swift）、`internal/app`（pdf_ops 等）、`internal/domain`（watermark/pdf_tools）。
> **结论**：P0×1、P1×5 全部修复；新增 4 组回归测试；`go test ./...` + `go vet` + `go build` 全绿。

---

## 一、P0：Zip Slip 目录逃逸（安全）

`ExtractVirtualItem` 用 `filepath.Join(dstDir, base+rel)` 拼接目标路径，压缩包内 `a/../../evil` 条目经 `Clean` 解析 `..` 后**逃逸出目标目录**（可写 `~/.zshenv` 等）。Go 的 `archive/zip` 不防御此问题。

**修复**（`internal/fs/archive_extract.go`）：新增 `safeJoinUnder(root, rel)`——先 `Abs` 再校验结果必须位于 root 之内，否则拒绝并报错。测试 `TestExtractVirtualItem_ZipSlipRejected` 构造恶意 zip 锁定行为。

## 二、P1 修复明细

1. **非 PNG/JPG 图片水印输出损坏**：`keep`（默认值）对 HEIC/WEBP/GIF/BMP/TIFF 输入按 JPEG 编码但保留原扩展名 → 生成 `.heic` 后缀的实际 JPEG（数据错乱）；部分格式编码返回 nil 被 `try?` 吞掉 → 静默无输出。
   **修复**：`domain.ResolveWatermarkOutputExt`（单一真相，逐文件预计算目标扩展名——keep 仅对 png/jpg 生效，其余一律转 png）；Swift 侧 `data == nil` 与写入失败均打 `CLOVER_SKIP` 由 Go 统计上报，不再静默。测试 `TestResolveWatermarkOutputExt`。
2. **tar/tar.gz 只有"进入"没有"内容"**：QuickLook 临时提取与包内右键提取原先只走 `zip.OpenReader`，tar 家族静默失败；且 tar 路径零测试覆盖。
   **修复**：`openTarReader` 统一 tar/tar.gz/tgz 打开；`extractTarEntryToFile`/`extractVirtualFromTar` 补齐两条链路（跳过符号链接等非常规条目）；测试 `TestArchiveVFS_TarGz` 覆盖目录钻取 + QuickLook 提取 + 落盘提取。
3. **Swift 源码拼接注入**：Go `%q` 产生 `\xNN`/`\uNNNN` 转义（如 NBSP→`\u00a0`），Swift 只认 `\u{XXXX}` → 水印文本含剪贴板粘贴的不换行空格时脚本编译失败。
   **修复**：新增 `swiftQuote`（`internal/macos/swift.go`，非打印字符输出 `\u{...}`），替换 pdf_service/watermark_service/finder_sidebar 全部 `%q` 拼接点。
4. **防覆盖缺失**：`merged.pdf`、`_watermarked` 固定名直接覆盖既有文件。**修复**：PDF 侧统一 `fs.NextAvailableName`；水印侧 Swift 端 `-2/-3` 序号探测。
5. **运行时依赖未披露**：PDF/水印全部依赖 `/usr/bin/swift`（Xcode CLT），未安装时功能失效且每次操作冷启动解释器。**现状接受 + 已在文档披露**；长远可换预编译 helper 或 `osascript -l JavaScript`。

## 三、其他修复

- 临时文件生命周期：QuickLook 暂存目录按压缩包路径隔离（不同包同名文件不再互相覆盖）+ 24h TTL 惰性清理 + 原子写（temp+rename，防半截预览文件）。
- SplitPDF 单页写失败原被 Swift `_ =` 吞掉 → 现在统计失败数 `exit(2)` 上报。
- 页数查询竞态：`ExecutePDFTools` 在解析范围前经 `refreshPageCount` 同步兜底。
- 合并菜单文案数量改为实际 PDF 选中数；`ExecuteFinderImport` 错误经 `FinderImportError` 入弹窗（模式与 PDFError/WatermarkError 统一）；`AddBookmarks` 去重键改 `filepath.Clean` 与 `AddBookmark` 对齐。
- banner"退出"改为与 ⌘↑ 一致的逐层退栈；UI 层 `📦💼💾⚙️🔖` emoji 清理（遵守 icons.go Zero Emoji policy）。
- 红线收敛：`ApplyWatermark`(109)/`readZipDir`(74)/`readTarStream`(75)/`ParsePageRanges`(71)/`buildWatermarkScript` 等全部拆分至 ≤50（脚本模板提为包级常量）；`archive_vfs.go` 拆出 `archive_extract.go`。剩余存量 UI 渲染函数（View/renderSinglePane/renderBatchRenameModal 等 51~72 行）留待专项治理。

## 四、已知残留（未修复，需后续决策）

1. tar 每次列目录全量流式解压（大包导航卡顿，无缓存）。
2. `StripEXIF` 语义由"重编码副作用"承担（效果成立），无独立开关。
3. `finder_sidebar.go` 保留 `.agy_home` 自动检测，新增 `CLOVER_USER_HOME` 环境变量注入口。
4. 需真机手测：VFS banner、工作区胶囊、PDF/水印/导入弹窗全流程。

## 五、新增回归测试

- `TestArchiveVFS_TarGz`：tar.gz 目录钻取 + QuickLook 提取 + 落盘提取。
- `TestExtractVirtualItem_ZipSlipRejected`：恶意 `../` 条目被拒绝且不落盘。
- `TestExtractArchiveFileToTemp_Isolated`：跨压缩包同名临时文件互不覆盖。
- `TestResolveWatermarkOutputExt`：13 组格式分流用例（含 iPhone HEIC 默认路径）。

## 六、优化轮（同日跟进）

1. **Swift 预编译 helper CLI**（`tools/clover-helper/main.swift` + `internal/macos/helper.go`）：PDF/水印/sfl3 全部改为 argv 参数化子命令调用，删除 Go 侧全部 Swift 源码拼接（含 swiftQuote）；探测顺序 环境变量 → 主程序同目录（.app 内）→ 仓库 build/ → `swift main.swift` 解释回退。收益：macos 包测试 4.5s→0.5s、运行时操作去冷启动、不再依赖 Xcode CLT、argv 传参彻底消除转义注入面。`make helper`/`make test`/`package-arm64.sh` 已接线。
2. **归档目录树缓存**（`internal/fs/archive_cache.go`）：key 含 mtime+size 自然失效，64 条 FIFO 淘汰；返回防御性副本避免调用方排序污染缓存。tar 大包重复导航从全量流式解压降为 O(1)。
3. **水印弹窗补齐角度/字号控件**（此前字段存在但 UI 未暴露，报告宣称的“旋转”才真正可用）；选项行抽公共 `renderWmOptionRow`，renderWatermarkBody 80→≤50 行。
4. **QuickLook 临时文件启动清理**：`main.go` 启动即 `go fs.CleanupQuickLookTemp()`。
5. 卫生：`🖼️⚠️ℹ️` emoji 全部清理为文字；测试夹具（测试 PDF/图片生成）收敛进 helper `make-test-pdf`/`make-test-image` 子命令，Go 测试不再拼接 Swift 源码。
