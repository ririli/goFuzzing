package breakpoint

import (
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ============================================================
// ParseSusPairs — 核心解析逻辑
// ============================================================

func TestParseSusPairs_SinglePair(t *testing.T) {
	c := NewConfig()
	c.ParseSusPairs("(1,2)")

	if !c.IsActive(1) {
		t.Error("funcId 1 should be active")
	}
	if !c.IsActive(2) {
		t.Error("funcId 2 should be active")
	}
	if c.IsActive(3) {
		t.Error("funcId 3 should NOT be active")
	}
	if got := c.FindPrev(2); len(got) != 1 || got[0] != 1 {
		t.Errorf("funcId 2 should have preId=[1], got %v", got)
	}
	if got := c.FindPrev(1); got != nil {
		t.Errorf("funcId 1 should have no preId, got %v", got)
	}
	if !c.DoWait(2) {
		t.Error("funcId 2 should DoWait")
	}
	if c.DoWait(1) {
		t.Error("funcId 1 should NOT DoWait")
	}
}

func TestParseSusPairs_MultiplePairs(t *testing.T) {
	c := NewConfig()
	c.ParseSusPairs("(1,2)(3,4)(5,6)")

	if !c.IsActive(1) || !c.IsActive(2) || !c.IsActive(3) ||
		!c.IsActive(4) || !c.IsActive(5) || !c.IsActive(6) {
		t.Error("all six funcIds should be active")
	}
	if got := c.FindPrev(2); len(got) != 1 || got[0] != 1 {
		t.Errorf("2 should wait for 1, got %v", got)
	}
	if got := c.FindPrev(4); len(got) != 1 || got[0] != 3 {
		t.Errorf("4 should wait for 3, got %v", got)
	}
}

// 同一后驱等待多个前驱
func TestParseSusPairs_SamePostMultiPre(t *testing.T) {
	c := NewConfig()
	c.ParseSusPairs("(1,3)(2,3)")

	prevs := c.FindPrev(3)
	if len(prevs) != 2 {
		t.Fatalf("funcId 3 should have 2 preIds, got %d", len(prevs))
	}
	if !c.DoWait(3) {
		t.Error("funcId 3 should DoWait")
	}
}

// 后驱本身也可以是另一个对的前驱
func TestParseSusPairs_Chain(t *testing.T) {
	c := NewConfig()
	c.ParseSusPairs("(1,2)(2,3)")

	if !c.IsActive(1) || !c.IsActive(2) || !c.IsActive(3) {
		t.Error("all three should be active")
	}
	if got := c.FindPrev(2); len(got) != 1 || got[0] != 1 {
		t.Errorf("2 should wait for 1")
	}
	if got := c.FindPrev(3); len(got) != 1 || got[0] != 2 {
		t.Errorf("3 should wait for 2")
	}
}

func TestParseSusPairs_EmptyString(t *testing.T) {
	c := NewConfig()
	c.ParseSusPairs("")
	if c.IsActive(1) {
		t.Error("no func should be active for empty input")
	}
}

func TestParseSusPairs_MalformedInput(t *testing.T) {
	c := NewConfig()
	c.ParseSusPairs("not-valid(a,b)(,)(123,")
	// 不应 panic，且不应有活跃函数
	if c.IsActive(1) {
		t.Error("no func should be active for malformed input")
	}
}

// ============================================================
// FindPrev / IsActive / DoWait / HasActive — 查找与状态
// ============================================================

func TestFindPrev_NotFound(t *testing.T) {
	c := NewConfig()
	if c.FindPrev(999) != nil {
		t.Error("unregistered funcId should return nil")
	}
}

func TestIsActive_False(t *testing.T) {
	c := NewConfig()
	if c.IsActive(42) {
		t.Error("unknown funcId should not be active")
	}
}

func TestDoWait_NotInWaitMap(t *testing.T) {
	c := NewConfig()
	if c.DoWait(1) {
		t.Error("funcId not in waitMap should not wait")
	}
}

func TestHasActive_InitiallyFalse(t *testing.T) {
	c := NewConfig()
	if c.HasActive() {
		t.Error("new Config should not have active")
	}
}

// ============================================================
// WaitMapDec — 等待计数递减
// ============================================================

func TestWaitMapDec_Normal(t *testing.T) {
	c := NewConfig()
	c.ParseSusPairs("(1,2)")
	atomic.StoreUint32(&c.hasActive, 1)
	if !c.DoWait(2) {
		t.Fatal("should wait initially")
	}
	c.WaitMapDec(2)
	if c.DoWait(2) {
		t.Error("should not wait after dec to 0")
	}
}

func TestWaitMapDec_MultiplePre(t *testing.T) {
	c := NewConfig()
	c.ParseSusPairs("(1,3)(2,3)")
	atomic.StoreUint32(&c.hasActive, 1)
	if !c.DoWait(3) {
		t.Fatal("should wait initially")
	}
	c.WaitMapDec(3) // still >0 (count was 2, now 1)
	if !c.DoWait(3) {
		t.Error("should still wait after one dec")
	}
	c.WaitMapDec(3) // now 0
	if c.DoWait(3) {
		t.Error("should not wait after second dec")
	}
}

func TestWaitMapDec_NotInMap(t *testing.T) {
	c := NewConfig()
	// 不应 panic
	c.WaitMapDec(999)
}

// ============================================================
// GetWaiter — channel 池
// ============================================================

func TestGetWaiter_SameIdReturnsSameChannel(t *testing.T) {
	c := NewConfig()
	w1 := c.GetWaiter(42)
	w2 := c.GetWaiter(42)
	if w1 != w2 {
		t.Error("same id should return same channel")
	}
}

func TestGetWaiter_DifferentIds(t *testing.T) {
	c := NewConfig()
	w1 := c.GetWaiter(1)
	w2 := c.GetWaiter(2)
	if w1 == w2 {
		t.Error("different ids should return different channels")
	}
}

// ============================================================
// CompleteOperation — 通知等待者
// ============================================================

func TestCompleteOperation_NotifiesWaiter(t *testing.T) {
	c := NewConfig()
	w := c.GetWaiter(100)
	// CompleteOperation should close the channel
	c.CompleteOperation(100)

	select {
	case <-w:
		// ok
	default:
		t.Error("channel should be closed after CompleteOperation")
	}
}

// 前向引用：CompleteOperation 发生在 GetWaiter 之前
func TestCompleteOperation_BeforeGetWaiter(t *testing.T) {
	c := NewConfig()
	c.CompleteOperation(200)
	w := c.GetWaiter(200)

	select {
	case <-w:
		// ok — 预先关闭的 channel 被返回
	default:
		t.Error("forward-reference channel should be pre-closed")
	}
}

func TestCompleteOperation_Idempotent(t *testing.T) {
	c := NewConfig()
	c.GetWaiter(1)
	c.CompleteOperation(1)
	// 第二次调用不应 panic
	c.CompleteOperation(1)
}

// ============================================================
// PointControl — 单栏断点控制
// ============================================================

func TestPointControl_NoActive(t *testing.T) {
	c := NewConfig()
	// 无活跃配置，应直接返回不阻塞
	c.PointControl(1)
	// 不应 panic
}

func TestPointControl_NotActiveFunc(t *testing.T) {
	c := NewConfig()
	c.ParseSusPairs("(1,2)")
	atomic.StoreUint32(&c.hasActive, 1)
	// funcId 99 不在活跃列表
	c.PointControl(99)
}

func TestPointControl_NoNeedToWait(t *testing.T) {
	c := NewConfig()
	c.Timeout = 10 * time.Millisecond
	c.ParseSusPairs("(1,2)")
	atomic.StoreUint32(&c.hasActive, 1)
	// funcId 1 是前驱，不需要等待
	c.PointControl(1)
}

func TestPointControl_WaitTimeout(t *testing.T) {
	c := NewConfig()
	c.Timeout = 10 * time.Millisecond
	c.ParseSusPairs("(1,2)")
	atomic.StoreUint32(&c.hasActive, 1)

	// funcId 2 等待 1，但 1 从未完成 → 超时
	c.PointControl(2)
	// 不应死锁，超时后应有 {TIMEOUT} 输出但函数正常返回
}

func TestPointControl_Covered(t *testing.T) {
	c := NewConfig()
	c.Timeout = 200 * time.Millisecond
	c.ParseSusPairs("(1,2)")
	atomic.StoreUint32(&c.hasActive, 1)

	var wg sync.WaitGroup
	var covered atomic.Bool

	wg.Add(1)
	go func() {
		defer wg.Done()
		// 1 先到达，执行后通过 CompleteOperation 通知
		c.PointControl(1)
		covered.Store(true)
	}()

	// 给 goroutine 一点时间先跑
	time.Sleep(10 * time.Millisecond)

	// 2 等待 1
	c.PointControl(2)
	wg.Wait()

	if !covered.Load() {
		t.Error("funcId 1 should have completed")
	}
}

func TestPointControl_MultiPre(t *testing.T) {
	c := NewConfig()
	c.Timeout = 200 * time.Millisecond
	c.ParseSusPairs("(1,3)(2,3)")
	atomic.StoreUint32(&c.hasActive, 1)

	var wg sync.WaitGroup

	// 两个前驱并发执行
	for _, id := range []uint64{1, 2} {
		wg.Add(1)
		go func(fid uint64) {
			defer wg.Done()
			time.Sleep(5 * time.Millisecond)
			c.PointControl(fid)
		}(id)
	}

	time.Sleep(10 * time.Millisecond)
	c.PointControl(3) // 等待 1 和 2
	wg.Wait()
}

// ============================================================
// 并发安全
// ============================================================

func TestConcurrentGetWaiter(t *testing.T) {
	c := NewConfig()
	var wg sync.WaitGroup
	n := 50
	wg.Add(n)
	chs := make([]chan struct{}, n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			chs[i] = c.GetWaiter(1)
		}(i)
	}
	wg.Wait()
	// 所有返回的 channel 应为同一个
	for i := 1; i < n; i++ {
		if chs[i] != chs[0] {
			t.Fatal("concurrent GetWaiter should return same channel")
		}
	}
}

func TestConcurrentParseSusPairs(t *testing.T) {
	c := NewConfig()
	var wg sync.WaitGroup
	n := 20
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			c.ParseSusPairs("(1,2)")
		}(i)
	}
	wg.Wait()
	// 不应 panic，且 2 应有前驱 1
	if got := c.FindPrev(2); len(got) == 0 || got[0] != 1 {
		t.Error("ParseSusPairs should survive concurrent calls")
	}
}

// ============================================================
// ParseInput — 环境变量入口
// ============================================================

func TestParseInput_Valid(t *testing.T) {
	c := NewConfig()
	os.Setenv("Input", "(10,20)")
	defer os.Unsetenv("Input")

	c.ParseInput()
	if !c.HasActive() {
		t.Error("HasActive should be true")
	}
	if !c.IsActive(10) || !c.IsActive(20) {
		t.Error("both funcIds should be active")
	}
}

func TestParseInput_Empty(t *testing.T) {
	c := NewConfig()
	os.Setenv("Input", "")
	defer os.Unsetenv("Input")

	c.ParseInput()
	if c.HasActive() {
		t.Error("empty Input should not activate")
	}
}

func TestParseInput_EnvNotSet(t *testing.T) {
	c := NewConfig()
	os.Unsetenv("Input")

	c.ParseInput()
	if c.HasActive() {
		t.Error("unset Input should not activate")
	}
}
