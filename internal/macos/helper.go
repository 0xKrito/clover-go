package macos

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// runHelper 执行原生能力 helper（PDF/水印/Finder 侧边栏）。
// 探测顺序：
//  1. 环境变量 CLOVER_HELPER_BIN（测试/开发显式指定）
//  2. 主程序同目录的 clover-helper（make bundle 放入 .app/Contents/MacOS）
//  3. 仓库 build/clover-helper（开发树）
//  4. 回退：用 /usr/bin/swift 解释 tools/clover-helper/main.swift（冷启动慢但可用）
//
// argv 传参彻底避免了源码拼接转义问题（文件名/水印文本可含任意字符）。
func runHelper(cmd string, args ...string) (string, error) {
	argv := append([]string{cmd}, args...)
	if bin, ok := findHelperBinary(); ok {
		return execProcess(bin, argv)
	}
	if src, ok := findHelperSource(); ok {
		return execProcess("/usr/bin/swift", append([]string{src}, argv...))
	}
	return "", fmt.Errorf("clover-helper 未找到：请执行 make helper 或 make bundle")
}

// findHelperBinary locates a precompiled helper binary.
func findHelperBinary() (string, bool) {
	var candidates []string
	if env := os.Getenv("CLOVER_HELPER_BIN"); env != "" {
		candidates = append(candidates, env)
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "clover-helper"))
	}
	if root, ok := findRepoRoot(); ok {
		candidates = append(candidates, filepath.Join(root, "build", "clover-helper"))
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c, true
		}
	}
	return "", false
}

// findHelperSource locates the helper Swift source for interpreted fallback.
func findHelperSource() (string, bool) {
	root, ok := findRepoRoot()
	if !ok {
		return "", false
	}
	p := filepath.Join(root, "tools", "clover-helper", "main.swift")
	if _, err := os.Stat(p); err != nil {
		return "", false
	}
	return p, true
}

// findRepoRoot walks up to the directory containing go.mod (go test's cwd is
// the package directory, not the repo root).
func findRepoRoot() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func execProcess(name string, argv []string) (string, error) {
	cmd := exec.Command(name, argv...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("%s failed: %s (%w)", filepath.Base(name), stderr.String(), err)
	}
	return stdout.String(), nil
}
