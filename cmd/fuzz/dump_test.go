package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"toolkit/pkg/bug"
)

func readFileForDumpTest(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func TestDumpBugReports_ClassifiesWithSource(t *testing.T) {
	dir := t.TempDir()
	bugs := bug.NewSet()
	bugs.Add(bug.Event{Kind: bug.KindPanic, Signature: "p1", Message: "panic: boom", Report: "panic: boom\nstack line"},
		bug.Evidence{Bin: "binA.exe", Fn: "TestA_1"})
	bugs.Add(bug.Event{Kind: bug.KindDataRace, Signature: "r1", Message: "race", Report: "WARNING: DATA RACE\naccess frames"},
		bug.Evidence{Bin: "binA.exe", Fn: "TestB_1"})
	bugs.Add(bug.Event{Kind: bug.KindFatal, Signature: "f1", Message: "fatal", Report: "fatal error: x"},
		bug.Evidence{})

	if err := dumpBugReports(bugs, dir); err != nil {
		t.Fatalf("dumpBugReports() error = %v", err)
	}

	panicContent := readFileForDumpTest(t, filepath.Join(dir, "allpanic.txt"))
	raceContent := readFileForDumpTest(t, filepath.Join(dir, "alldatarace.txt"))

	// panic 报告含完整 report 与来源定位
	if !strings.Contains(panicContent, "panic: boom") ||
		!strings.Contains(panicContent, "bin=binA.exe") ||
		!strings.Contains(panicContent, "test=TestA_1") {
		t.Errorf("allpanic.txt missing panic record or source info:\n%s", panicContent)
	}
	// 两个文件互不串扰
	if strings.Contains(panicContent, "WARNING: DATA RACE") {
		t.Error("allpanic.txt must not contain data race records")
	}
	if !strings.Contains(raceContent, "WARNING: DATA RACE") || !strings.Contains(raceContent, "test=TestB_1") {
		t.Errorf("alldatarace.txt missing race record or source info:\n%s", raceContent)
	}
	if strings.Contains(raceContent, "panic: boom") {
		t.Error("alldatarace.txt must not contain panic records")
	}
	// fatal 等其他种类不写入聚合文件
	if strings.Contains(panicContent, "fatal error") || strings.Contains(raceContent, "fatal error") {
		t.Error("fatal records must not be written into aggregated files")
	}
}

func TestDumpBugReports_AppendsAcrossRuns(t *testing.T) {
	dir := t.TempDir()
	bugs := bug.NewSet()
	bugs.Add(bug.Event{Kind: bug.KindPanic, Signature: "p1", Message: "panic: boom", Report: "panic: boom"},
		bug.Evidence{Bin: "binA.exe", Fn: "TestA_1"})

	// 模拟脚本按二进制逐个调用 fuzz：同一目录多次 dump 应追加累积
	for i := 0; i < 2; i++ {
		if err := dumpBugReports(bugs, dir); err != nil {
			t.Fatalf("dumpBugReports() round %d error = %v", i, err)
		}
	}

	panicContent := readFileForDumpTest(t, filepath.Join(dir, "allpanic.txt"))
	if n := strings.Count(panicContent, "======== panic #1 ========"); n != 2 {
		t.Errorf("panic record sections = %d, want 2 (append mode)", n)
	}
}

func TestDumpBugReports_EmptySetStillCreatesFiles(t *testing.T) {
	dir := t.TempDir()
	if err := dumpBugReports(bug.NewSet(), dir); err != nil {
		t.Fatalf("dumpBugReports() error = %v", err)
	}
	for _, name := range []string{"allpanic.txt", "alldatarace.txt"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s should exist even when no records collected: %v", name, err)
		}
	}
}
