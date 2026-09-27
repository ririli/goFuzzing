package function

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// resetTestConfig 重置包级 cfg，避免测试间状态污染。
func resetTestConfig(timeout time.Duration) {
	cfg = &Config{
		barriers:       make(map[uint64][]*barrierGate),
		activeMap:      make(map[uint64]struct{}),
		BarrierTimeout: timeout,
	}
}

// ---------- ParsePairs ----------

func TestParsePairs_SetsActiveAndBarriers(t *testing.T) {
	resetTestConfig(10 * time.Millisecond)

	ParsePairs("(1,2)(3,4)")

	if len(cfg.activeMap) != 4 {
		t.Fatalf("activeMap len = %d, want 4", len(cfg.activeMap))
	}
	for _, id := range []uint64{1, 2, 3, 4} {
		if _, ok := cfg.activeMap[id]; !ok {
			t.Errorf("activeMap missing id %d", id)
		}
	}
	if len(cfg.barriers[1]) != 1 || len(cfg.barriers[2]) != 1 {
		t.Error("pair (1,2) should register one shared gate on both sides")
	}
	if cfg.barriers[1][0] != cfg.barriers[2][0] {
		t.Error("both sides must share the same barrierGate")
	}
}

func TestParsePairs_IgnoresMalformed(t *testing.T) {
	resetTestConfig(10 * time.Millisecond)

	// 非数字 ID 与缺少右括号的对都应被丢弃
	ParsePairs("(abc,def)(3")

	if len(cfg.activeMap) != 0 {
		t.Errorf("activeMap len = %d, want 0 for malformed input", len(cfg.activeMap))
	}
	if len(cfg.barriers) != 0 {
		t.Errorf("barriers len = %d, want 0 for malformed input", len(cfg.barriers))
	}
}

func TestHasActive(t *testing.T) {
	resetTestConfig(10 * time.Millisecond)

	if HasActive() {
		t.Error("HasActive() should be false before any input")
	}
	atomic.StoreUint32(&cfg.hasActive, 1)
	if !HasActive() {
		t.Error("HasActive() should be true after hasActive set")
	}
}

// ---------- PointControl ----------

func TestPointControl_NoActiveReturnsImmediately(t *testing.T) {
	resetTestConfig(time.Second)
	// hasActive=0 时必须立即返回，不阻塞
	done := make(chan struct{})
	go func() {
		PointControl(1)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("PointControl blocked without active input")
	}
}

func TestPointControl_InactiveIDReturnsImmediately(t *testing.T) {
	resetTestConfig(time.Second)
	atomic.StoreUint32(&cfg.hasActive, 1)
	ParsePairs("(1,2)")

	done := make(chan struct{})
	go func() {
		PointControl(999) // 未调度的 ID
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("PointControl blocked for an unscheduled funcID")
	}
}

func TestPointControl_RendezvousReleasesBoth(t *testing.T) {
	resetTestConfig(time.Second)
	atomic.StoreUint32(&cfg.hasActive, 1)
	ParsePairs("(10,20)")

	var wg sync.WaitGroup
	wg.Add(2)
	start := time.Now()
	go func() { defer wg.Done(); PointControl(10) }()
	go func() { defer wg.Done(); PointControl(20) }()

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
		if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
			t.Errorf("rendezvous took %v, want prompt release", elapsed)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("rendezvous deadlocked: both arrivals did not release")
	}

	gate := cfg.barriers[10][0]
	if atomic.LoadInt32(&gate.arrived) != 3 {
		t.Errorf("gate.arrived = %d, want 3", gate.arrived)
	}
	if atomic.LoadInt32(&gate.expired) != 0 {
		t.Error("gate should not expire on successful rendezvous")
	}
}

func TestPointControl_SingleArrivalTimesOut(t *testing.T) {
	resetTestConfig(30 * time.Millisecond)
	atomic.StoreUint32(&cfg.hasActive, 1)
	ParsePairs("(10,20)")

	start := time.Now()
	PointControl(10) // 伙伴永远不到达
	elapsed := time.Since(start)

	if elapsed < 30*time.Millisecond {
		t.Errorf("first arrival returned after %v, want to wait BarrierTimeout", elapsed)
	}
	gate := cfg.barriers[10][0]
	if atomic.LoadInt32(&gate.expired) != 1 {
		t.Error("gate should be marked expired after timeout")
	}
}

func TestPointControl_LateArrivalAfterExpiryDoesNotDeadlock(t *testing.T) {
	resetTestConfig(20 * time.Millisecond)
	atomic.StoreUint32(&cfg.hasActive, 1)
	ParsePairs("(10,20)")

	PointControl(10) // 超时后返回，gate 标记 expired

	done := make(chan struct{})
	go func() {
		PointControl(20) // 迟到者：不得阻塞或 panic
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("late arrival after expiry blocked")
	}
}
