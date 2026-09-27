package audit

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"toolkit/pkg/bug"
	"toolkit/pkg/fuzzer"
	"toolkit/pkg/operation"
)

func TestOperationWaiterWakesAfterPredecessor(t *testing.T) {
	operation.ParsePair("(10001,10002)(10001,10003)")
	done := make(chan struct{}, 2)
	for _, id := range []uint64{10002, 10003} {
		go func(id uint64) { operation.InstChBF(id); done <- struct{}{} }(id)
	}
	time.Sleep(30 * time.Millisecond)
	operation.InstChAF(10001, make(chan int), "close")
	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(400 * time.Millisecond):
			t.Fatal("predecessor completion did not wake every waiting operation")
		}
	}
}

func TestAuditHelper_1(t *testing.T) {
	switch os.Getenv("GOPIE_AUDIT_HELPER") {
	case "hang":
		time.Sleep(30 * time.Second)
	case "output":
		fmt.Fprintln(os.Stderr, strings.Repeat("x", 256*1024))
		fmt.Fprintln(os.Stderr, "panic: audit-tail-marker")
	}
}

// This must not be selected when Monitor runs TestAuditHelper_1.
func TestAuditHelper_10(t *testing.T) {
	if os.Getenv("GOPIE_AUDIT_HELPER") != "" {
		fmt.Fprintln(os.Stderr, "panic: wrong-test-selected")
	}
}

func TestMonitorCapturesTailAndSelectsExactTest(t *testing.T) {
	t.Setenv("GOPIE_AUDIT_HELPER", "output")
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cfg := fuzzer.DefaultConfig()
	cfg.Bin, cfg.Fn = bin, "TestAuditHelper_1"
	cfg.MaxWorker, cfg.MaxExecution = 1, 1
	cfg.SharedBugs = bug.NewSet()
	failed, _ := (&fuzzer.Monitor{}).Start(cfg, nil)
	if !failed {
		t.Fatal("oracle after a long log line was lost")
	}
	records := cfg.SharedBugs.Snapshot()
	if len(records) != 1 || records[0].Message != "audit-tail-marker" {
		t.Fatalf("wrong test or incomplete output: %#v", records)
	}
}

func TestMonitorCancelsUnlimitedExecution(t *testing.T) {
	t.Setenv("GOPIE_AUDIT_HELPER", "hang")
	bin, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cfg := fuzzer.DefaultConfig()
	cfg.Bin, cfg.Fn = bin, "TestAuditHelper_1"
	cfg.TimeOut, cfg.MaxWorker, cfg.MaxFuzzTime = 0, 2, 1
	start := time.Now()
	(&fuzzer.Monitor{}).Start(cfg, nil)
	if time.Since(start) > 5*time.Second {
		t.Fatal("session cancellation did not reap workers")
	}
}
