package breakpoint

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ============================================================
// NewBarrierConfig — 构造函数
// ============================================================

func TestNewBarrierConfig(t *testing.T) {
	b := NewBarrierConfig()
	if b == nil {
		t.Fatal("NewBarrierConfig returned nil")
	}
	if b.Timeout != 40*time.Millisecond {
		t.Errorf("default timeout should be 40ms, got %v", b.Timeout)
	}
	if b.HasActive() {
		t.Error("new config should not be active")
	}
}

// ============================================================
// ParseSusPairs — 双栏解析
// ============================================================

func TestBarrierParseSusPairs_SinglePair(t *testing.T) {
	b := NewBarrierConfig()
	b.ParseSusPairs("(1,2)")

	if !b.IsActive(1) || !b.IsActive(2) {
		t.Error("both funcIds should be active")
	}
	if b.IsActive(3) {
		t.Error("funcId 3 should not be active")
	}
}

func TestBarrierParseSusPairs_MultiplePairs(t *testing.T) {
	b := NewBarrierConfig()
	b.ParseSusPairs("(1,2)(3,4)(5,6)")

	for _, id := range []uint64{1, 2, 3, 4, 5, 6} {
		if !b.IsActive(id) {
			t.Errorf("funcId %d should be active", id)
		}
	}
}

func TestBarrierParseSusPairs_OverlappingPairs(t *testing.T) {
	b := NewBarrierConfig()
	// 1 参与两个 barrier: (1,2) 和 (1,3)
	b.ParseSusPairs("(1,2)(1,3)")

	if !b.IsActive(1) || !b.IsActive(2) || !b.IsActive(3) {
		t.Error("all three should be active")
	}
}

func TestBarrierParseSusPairs_Empty(t *testing.T) {
	b := NewBarrierConfig()
	b.ParseSusPairs("")
	if b.IsActive(1) {
		t.Error("no func should be active for empty input")
	}
}

func TestBarrierParseSusPairs_Malformed(t *testing.T) {
	b := NewBarrierConfig()
	b.ParseSusPairs("not-valid(a,b)(,)(123,")
	if b.IsActive(1) {
		t.Error("no func should be active for malformed input")
	}
}

// ============================================================
// HasActive / IsActive — 状态查询
// ============================================================

func TestBarrierHasActive_InitiallyFalse(t *testing.T) {
	b := NewBarrierConfig()
	if b.HasActive() {
		t.Error("new BarrierConfig should not have active")
	}
}

func TestBarrierIsActive_Unknown(t *testing.T) {
	b := NewBarrierConfig()
	b.ParseSusPairs("(1,2)")
	if b.IsActive(99) {
		t.Error("unknown funcId should not be active")
	}
}

func TestBarrierIsActive_Concurrent(t *testing.T) {
	b := NewBarrierConfig()
	b.ParseSusPairs("(1,2)")
	atomic.StoreUint32(&b.hasActive, 1)

	var wg sync.WaitGroup
	n := 50
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			if !b.IsActive(1) {
				t.Error("funcId 1 should be active during concurrent reads")
			}
		}()
	}
	wg.Wait()
}

// ============================================================
// PointControl — 双栏断点控制（核心）
// ============================================================

func TestBarrierPointControl_NoActive(t *testing.T) {
	b := NewBarrierConfig()
	// 无活跃配置，直接返回
	b.PointControl(1)
}

func TestBarrierPointControl_NotActiveFunc(t *testing.T) {
	b := NewBarrierConfig()
	b.ParseSusPairs("(1,2)")
	atomic.StoreUint32(&b.hasActive, 1)
	// funcId 99 不在配置中
	b.PointControl(99)
}

func TestBarrierPointControl_Covered(t *testing.T) {
	b := NewBarrierConfig()
	b.Timeout = 200 * time.Millisecond
	b.ParseSusPairs("(1,2)")
	atomic.StoreUint32(&b.hasActive, 1)

	var wg sync.WaitGroup
	var arrived atomic.Bool

	wg.Add(1)
	go func() {
		defer wg.Done()
		// 模拟 funcId 1 到达入口
		b.PointControl(1)
		arrived.Store(true)
	}()

	// 给 goroutine 时间先到达
	time.Sleep(10 * time.Millisecond)

	// funcId 2 到达 —— 两者都到达，同时放行
	b.PointControl(2)

	wg.Wait()
	if !arrived.Load() {
		t.Error("funcId 1 should have been released")
	}
}

func TestBarrierPointControl_Timeout(t *testing.T) {
	b := NewBarrierConfig()
	b.Timeout = 10 * time.Millisecond
	b.ParseSusPairs("(1,2)")
	atomic.StoreUint32(&b.hasActive, 1)

	// 只有 1 到达，2 永不出现 → 超时
	// 不应死锁
	b.PointControl(1)
}

func TestBarrierPointControl_TwoPairsOneFunc(t *testing.T) {
	b := NewBarrierConfig()
	b.Timeout = 200 * time.Millisecond
	b.ParseSusPairs("(1,2)(1,3)")
	atomic.StoreUint32(&b.hasActive, 1)

	var wg sync.WaitGroup
	var arrived2, arrived3 atomic.Bool

	wg.Add(1)
	go func() {
		defer wg.Done()
		b.PointControl(2) // 先到达第一个 barrier
		arrived2.Store(true)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		b.PointControl(3) // 先到达第二个 barrier
		arrived3.Store(true)
	}()

	time.Sleep(10 * time.Millisecond)

	// 1 到达 — 同时释放两个 barrier
	b.PointControl(1)

	wg.Wait()
	if !arrived2.Load() {
		t.Error("funcId 2 should be released")
	}
	if !arrived3.Load() {
		t.Error("funcId 3 should be released")
	}
}

func TestBarrierPointControl_SecondArriverDoesNotBlock(t *testing.T) {
	b := NewBarrierConfig()
	b.Timeout = 50 * time.Millisecond
	b.ParseSusPairs("(1,2)")
	atomic.StoreUint32(&b.hasActive, 1)

	done := make(chan struct{})
	go func() {
		b.PointControl(1) // 第一个到达 → 阻塞
		close(done)
	}()

	time.Sleep(10 * time.Millisecond)

	// 第二个到达不应该阻塞
	start := time.Now()
	b.PointControl(2) // 同时释放
	elapsed := time.Since(start)

	<-done

	if elapsed > 50*time.Millisecond {
		t.Errorf("second arriver blocked too long: %v", elapsed)
	}
}

// ============================================================
// 并发安全
// ============================================================

func TestBarrierConcurrentParseSusPairs(t *testing.T) {
	b := NewBarrierConfig()
	var wg sync.WaitGroup
	n := 20
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			b.ParseSusPairs("(1,2)")
		}()
	}
	wg.Wait()
	if !b.IsActive(1) || !b.IsActive(2) {
		t.Error("concurrent ParseSusPairs should succeed")
	}
}

func TestBarrierConcurrentPointControl(t *testing.T) {
	b := NewBarrierConfig()
	b.Timeout = 500 * time.Millisecond
	b.ParseSusPairs("(1,2)")
	atomic.StoreUint32(&b.hasActive, 1)

	var wg sync.WaitGroup
	n := 10
	var success atomic.Int32

	for i := 0; i < n; i++ {
		wg.Add(2)
		// 每一对 (1,2) 在两个 goroutine 中同时执行
		go func() {
			defer wg.Done()
			b.PointControl(1)
			success.Add(1)
		}()
		go func() {
			defer wg.Done()
			b.PointControl(2)
			success.Add(1)
		}()
	}

	wg.Wait()
	if success.Load() != int32(2*n) {
		t.Errorf("expected %d completions, got %d", 2*n, success.Load())
	}
}

// ============================================================
// ParseInput — 环境变量入口
// ============================================================

func TestBarrierParseInput_Valid(t *testing.T) {
	b := NewBarrierConfig()
	os.Setenv("Input", "(10,20)")
	defer os.Unsetenv("Input")

	b.ParseInput()
	if !b.HasActive() {
		t.Error("should be active after ParseInput")
	}
	if !b.IsActive(10) || !b.IsActive(20) {
		t.Error("both funcIds should be active")
	}
}

func TestBarrierParseInput_Empty(t *testing.T) {
	b := NewBarrierConfig()
	os.Setenv("Input", "")
	defer os.Unsetenv("Input")

	b.ParseInput()
	if b.HasActive() {
		t.Error("empty Input should not activate")
	}
}

func TestBarrierParseInput_Unset(t *testing.T) {
	b := NewBarrierConfig()
	os.Unsetenv("Input")

	b.ParseInput()
	if b.HasActive() {
		t.Error("unset Input should not activate")
	}
}
