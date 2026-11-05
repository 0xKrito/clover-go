import Foundation
import PDFKit
import AppKit
import AVFoundation

// CloverGo 原生能力 helper。
// - 预编译：swiftc -O -o build/clover-helper tools/clover-helper/main.swift
//   （make bundle 自动执行，二进制随 .app 分发，运行时无需 Xcode CLT）
// - 解释执行（开发/测试回退）：swift tools/clover-helper/main.swift <cmd> ...
//
// 约定：结果走 stdout；失败信息走 stderr + 非零退出；水印单文件失败输出
// CLOVER_SKIP 行，由 Go 调用方统计上报。

func fail(_ msg: String, _ code: Int32 = 1) -> Never {
    fputs(msg + "\n", stderr)
    exit(code)
}

let argv = CommandLine.arguments
guard argv.count >= 2 else { fail("usage: clover-helper <cmd> [args...]") }
let cmd = argv[1]
let rest = Array(argv.dropFirst(2))

func parsePages(_ csv: String) -> [Int] {
    return csv.split(separator: ",").compactMap { Int($0.trimmingCharacters(in: .whitespaces)) }
}

switch cmd {
case "pdf-pagecount":
    guard rest.count == 1, let doc = PDFDocument(url: URL(fileURLWithPath: rest[0])) else {
        fail("cannot open pdf")
    }
    print(doc.pageCount)

case "pdf-extract":
    guard rest.count == 3, let src = PDFDocument(url: URL(fileURLWithPath: rest[0])) else {
        fail("cannot open src pdf")
    }
    let out = PDFDocument()
    for p in parsePages(rest[2]) where p >= 1 && p <= src.pageCount {
        if let page = src.page(at: p - 1) {
            out.insert(page, at: out.pageCount)
        }
    }
    guard out.write(to: URL(fileURLWithPath: rest[1])) else {
        fail("failed to write output pdf", 2)
    }

case "pdf-split":
    guard rest.count == 2, let src = PDFDocument(url: URL(fileURLWithPath: rest[0])) else {
        fail("cannot open src pdf")
    }
    let outDir = rest[1]
    let baseName = URL(fileURLWithPath: rest[0]).deletingPathExtension().lastPathComponent
    var failed = 0
    for i in 0..<src.pageCount {
        let out = PDFDocument()
        if let page = src.page(at: i) {
            out.insert(page, at: 0)
            let pageNum = String(format: "%03d", i + 1)
            let outPath = "\(outDir)/\(baseName)_page_\(pageNum).pdf"
            if !out.write(to: URL(fileURLWithPath: outPath)) {
                failed += 1
            }
        }
    }
    if failed > 0 {
        fail("failed to write \(failed) page file(s)", 2)
    }

case "pdf-export-images":
    guard rest.count == 5, let src = PDFDocument(url: URL(fileURLWithPath: rest[0])) else {
        fail("cannot open src pdf")
    }
    let outDir = rest[1]
    let isJpg = rest[2].lowercased() == "jpg" || rest[2].lowercased() == "jpeg"
    let dpi = Double(rest[3]) ?? 150
    let pages = parsePages(rest[4])
    let baseName = URL(fileURLWithPath: rest[0]).deletingPathExtension().lastPathComponent
    let scale = CGFloat(dpi) / 72.0

    for p in pages {
        guard p >= 1 && p <= src.pageCount, let page = src.page(at: p - 1) else { continue }
        let rect = page.bounds(for: .mediaBox)
        let w = max(1, Int(rect.width * scale))
        let h = max(1, Int(rect.height * scale))

        guard let rep = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: w, pixelsHigh: h, bitsPerSample: 8, samplesPerPixel: 4, hasAlpha: true, isPlanar: false, colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0) else { continue }

        NSGraphicsContext.saveGraphicsState()
        NSGraphicsContext.current = NSGraphicsContext(bitmapImageRep: rep)
        let ctx = NSGraphicsContext.current?.cgContext
        ctx?.setFillColor(CGColor.white)
        ctx?.fill(CGRect(x: 0, y: 0, width: w, height: h))
        ctx?.scaleBy(x: scale, y: scale)
        page.draw(with: .mediaBox, to: ctx!)
        NSGraphicsContext.restoreGraphicsState()

        let pageNum = String(format: "%03d", p)
        let ext = isJpg ? "jpg" : "png"
        let properties: [NSBitmapImageRep.PropertyKey: Any] = isJpg
            ? [.compressionFactor: 0.85]
            : [:]
        let type: NSBitmapImageRep.FileType = isJpg ? .jpeg : .png
        if let data = rep.representation(using: type, properties: properties) {
            let outUrl = URL(fileURLWithPath: "\(outDir)/\(baseName)_page_\(pageNum).\(ext)")
            try? data.write(to: outUrl)
        }
    }

case "make-test-pdf":
    // 测试辅助：生成 N 页的纯色测试 PDF（main.swift 自身能力，避免 Go 侧再拼 Swift 源码）。
    guard rest.count == 2, let pages = Int(rest[1]), pages > 0 else { fail("make-test-pdf: bad args") }
    let rep = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: 100, pixelsHigh: 100, bitsPerSample: 8, samplesPerPixel: 4, hasAlpha: true, isPlanar: false, colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0)!
    let img = NSImage(size: NSSize(width: 100, height: 100))
    img.addRepresentation(rep)
    let doc = PDFDocument()
    for _ in 0..<pages {
        if let p = PDFPage(image: img) {
            doc.insert(p, at: doc.pageCount)
        }
    }
    guard doc.write(to: URL(fileURLWithPath: rest[0])) else { fail("failed to write test pdf", 2) }

case "make-test-image":
    // 测试辅助：生成纯色测试图片（jpg/png）。
    guard rest.count == 4, let w = Int(rest[1]), let h = Int(rest[2]), w > 0, h > 0 else { fail("make-test-image: bad args") }
    let rep = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: w, pixelsHigh: h, bitsPerSample: 8, samplesPerPixel: 4, hasAlpha: true, isPlanar: false, colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0)!
    let img = NSImage(size: NSSize(width: w, height: h))
    img.addRepresentation(rep)
    guard let tiff = img.tiffRepresentation, let r = NSBitmapImageRep(data: tiff) else { fail("make-test-image: rep failed") }
    let type: NSBitmapImageRep.FileType = rest[3].lowercased() == "png" ? .png : .jpeg
    guard let data = r.representation(using: type, properties: [:]) else { fail("make-test-image: encode failed") }
    do {
        try data.write(to: URL(fileURLWithPath: rest[0]))
    } catch {
        fail("make-test-image: write failed")
    }

case "pdf-merge":
    guard rest.count >= 2 else { fail("no pdfs to merge") }
    let dst = rest[0]
    let sources = Array(rest.dropFirst())
    let out = PDFDocument()
    for path in sources {
        if let src = PDFDocument(url: URL(fileURLWithPath: path)) {
            for i in 0..<src.pageCount {
                if let page = src.page(at: i) {
                    out.insert(page, at: out.pageCount)
                }
            }
        }
    }
    guard out.write(to: URL(fileURLWithPath: dst)) else {
        fail("failed to write merged pdf", 2)
    }

case "watermark":
    // argv: text tile opacity angle fontSize r g b extsCSV files...
    guard rest.count >= 10 else { fail("watermark: missing args") }
    let watermarkText = rest[0]
    let isTile = rest[1] == "1"
    let opacity = CGFloat(Double(rest[2]) ?? 0.35)
    let angleDeg = CGFloat(Double(rest[3]) ?? -30)
    let fontSize = CGFloat(Double(rest[4]) ?? 24)
    let r = CGFloat(Double(rest[5]) ?? 0.5)
    let g = CGFloat(Double(rest[6]) ?? 0.5)
    let b = CGFloat(Double(rest[7]) ?? 0.5)
    let targetExts = rest[8].split(separator: ",").map(String.init)
    let filePaths = Array(rest.dropFirst(9))
    guard filePaths.count == targetExts.count else { fail("watermark: ext/file count mismatch") }

    let font = NSFont.systemFont(ofSize: fontSize, weight: .bold)
    let color = NSColor(calibratedRed: r, green: g, blue: b, alpha: opacity)
    let attrs: [NSAttributedString.Key: Any] = [
        .font: font,
        .foregroundColor: color
    ]
    let attrStr = NSAttributedString(string: watermarkText, attributes: attrs)
    let strSize = attrStr.size()
    let fm = FileManager.default

    for (idx, path) in filePaths.enumerated() {
        let targetExt = targetExts[idx]
        guard let img = NSImage(contentsOfFile: path) else {
            print("CLOVER_SKIP"); continue
        }
        let imgSize = img.size
        guard imgSize.width > 0, imgSize.height > 0 else {
            print("CLOVER_SKIP"); continue
        }

        let outImg = NSImage(size: imgSize)
        outImg.lockFocus()
        img.draw(at: .zero, from: NSRect(origin: .zero, size: imgSize), operation: .sourceOver, fraction: 1.0)
        let ctx = NSGraphicsContext.current?.cgContext
        drawWatermarkLayer(ctx, attrStr, strSize, imgSize, isTile, angleDeg)
        outImg.unlockFocus()

        let dir = URL(fileURLWithPath: path).deletingLastPathComponent().path
        let base = URL(fileURLWithPath: path).deletingPathExtension().lastPathComponent
        var outPath = "\(dir)/\(base)_watermarked.\(targetExt)"
        var seq = 2
        while fm.fileExists(atPath: outPath) {
            outPath = "\(dir)/\(base)_watermarked-\(seq).\(targetExt)"
            seq += 1
        }

        guard let tiff = outImg.tiffRepresentation, let rep = NSBitmapImageRep(data: tiff) else {
            print("CLOVER_SKIP"); continue
        }
        var data: Data? = nil
        if targetExt == "png" {
            data = rep.representation(using: .png, properties: [:])
        } else {
            data = rep.representation(using: .jpeg, properties: [.compressionFactor: 0.90])
        }
        if data == nil {
            print("CLOVER_SKIP"); continue
        }
        do {
            try data!.write(to: URL(fileURLWithPath: outPath))
        } catch {
            print("CLOVER_SKIP"); continue
        }
    }

case "finder-favs":
    guard rest.count == 1,
          let data = try? Data(contentsOf: URL(fileURLWithPath: rest[0])),
          let unarchived = try? NSKeyedUnarchiver.unarchiveTopLevelObjectWithData(data) as? NSDictionary,
          let items = unarchived["items"] as? [NSObject] else {
        fail("cannot read sfl3")
    }
    for item in items {
        if let bookmarkData = item.value(forKey: "Bookmark") as? Data {
            var isStale = false
            if let url = try? URL(resolvingBookmarkData: bookmarkData, options: .withoutUI, relativeTo: nil, bookmarkDataIsStale: &isStale) {
                let p = url.path
                if !p.isEmpty {
                    print("\(url.lastPathComponent)\t\(p)")
                }
            }
        }
    }

case "video-thumb":
    guard rest.count >= 2 else { fail("video-thumb: bad args") }
    let srcUrl = URL(fileURLWithPath: rest[0])
    let outUrl = URL(fileURLWithPath: rest[1])
    let maxPx = rest.count >= 3 ? (Double(rest[2]) ?? 512) : 512
    let asset = AVURLAsset(url: srcUrl)
    let gen = AVAssetImageGenerator(asset: asset)
    gen.appliesPreferredTrackTransform = true
    gen.maximumSize = CGSize(width: maxPx, height: maxPx)
    let time = CMTime(seconds: 1.0, preferredTimescale: 600)
    var extractedCG: CGImage? = nil
    if let cg = try? gen.copyCGImage(at: time, actualTime: nil) {
        extractedCG = cg
    } else if let cg = try? gen.copyCGImage(at: .zero, actualTime: nil) {
        extractedCG = cg
    }
    guard let cgImage = extractedCG else {
        fail("video-thumb: extract frame failed")
    }
    let rep = NSBitmapImageRep(cgImage: cgImage)
    guard let pngData = rep.representation(using: .png, properties: [:]) else {
        fail("video-thumb: encode png failed")
    }
    do {
        try pngData.write(to: outUrl)
    } catch {
        fail("video-thumb: write failed")
    }

default:
    fail("unknown command: \(cmd)")
}

func drawWatermarkLayer(_ ctx: CGContext?, _ attrStr: NSAttributedString, _ strSize: CGSize, _ imgSize: CGSize, _ isTile: Bool, _ angleDeg: CGFloat) {
    if isTile {
        let stepX = max(180, strSize.width * 1.5)
        let stepY = max(120, strSize.height * 4.0)
        var y: CGFloat = -imgSize.height * 0.2
        while y < imgSize.height * 1.2 {
            var x: CGFloat = -imgSize.width * 0.2
            while x < imgSize.width * 1.2 {
                ctx?.saveGState()
                ctx?.translateBy(x: x, y: y)
                ctx?.rotate(by: angleDeg * .pi / 180.0)
                attrStr.draw(at: NSPoint(x: -strSize.width / 2, y: -strSize.height / 2))
                ctx?.restoreGState()
                x += stepX
            }
            y += stepY
        }
    } else {
        ctx?.saveGState()
        ctx?.translateBy(x: imgSize.width / 2, y: imgSize.height / 2)
        ctx?.rotate(by: angleDeg * .pi / 180.0)
        attrStr.draw(at: NSPoint(x: -strSize.width / 2, y: -strSize.height / 2))
        ctx?.restoreGState()
    }
}
