package fuzzer

import (
	"sync/atomic"
	"testing"

	"toolkit/pkg/feedback"
)

// ---------- 键生成 ----------

func TestGortPairKey(t *testing.T) {
	loc1 := feedback.CallLocationInfo{File: "a.go", Line: 10}
	loc2 := feedback.CallLocationInfo{File: "b.go", Line: 20}

	// 小的 gid 应在前面
	pair := &feedback.GortPairInfo{
		Gid1: 100, Gid2: 50,
		CallLoc1: loc1, CallLoc2: loc2,
	}
	key := gortPairKey(pair)
	// 50-100，callLoc 对应 gid 顺序也要交换
	want := "50-100|b.go:20-a.go:10"
	if key != want {
		t.Errorf("gortPairKey() = %q, want %q", key, want)
	}

	// 已有序时不变
	pair2 := &feedback.GortPairInfo{
		Gid1: 10, Gid2: 20,
		CallLoc1: loc1, CallLoc2: loc2,
	}
	key2 := gortPairKey(pair2)
	want2 := "10-20|a.go:10-b.go:20"
	if key2 != want2 {
		t.Errorf("gortPairKey() = %q, want %q", key2, want2)
	}
}

func TestGortSignalKey(t *testing.T) {
	// 大的 id 放后面
	if got := gortSignalKey(100, 50); got != "50-100" {
		t.Errorf("gortSignalKey(100,50) = %q, want '50-100'", got)
	}
	if got := gortSignalKey(10, 20); got != "10-20" {
		t.Errorf("gortSignalKey(10,20) = %q, want '10-20'", got)
	}
	if got := gortSignalKey(5, 5); got != "5-5" {
		t.Errorf("gortSignalKey(5,5) = %q, want '5-5'", got)
	}
}

// ---------- 构造 ----------

func TestNewCorpusGort(t *testing.T) {
	var phase uint32
	cg := NewCorpusGort(&phase)

	if cg.CoveredConPairs == nil {
		t.Error("CoveredConPairs should be initialized")
	}
	if cg.SusConPairs == nil {
		t.Error("SusConPairs should be initialized")
	}
	if cg.InfeasiblePairs == nil {
		t.Error("InfeasiblePairs should be initialized")
	}
	if cg.TryPairs == nil {
		t.Error("TryPairs should be initialized")
	}
	if cg.pairTimeouts == nil {
		t.Error("pairTimeouts should be initialized")
	}
	if cg.selectNum != 1 {
		t.Errorf("selectNum = %d, want 1", cg.selectNum)
	}
	if cg.gortPhase != &phase {
		t.Error("gortPhase should point to phase")
	}
}

// ---------- AddPair ----------

func TestCorpusGort_AddPair_Observed(t *testing.T) {
	var phase uint32 = 0
	cg := NewCorpusGort(&phase)

	pair := &feedback.GortPairInfo{
		Gid1: 1, Gid2: 2,
		CallLoc1:   feedback.CallLocationInfo{File: "x.go", Line: 1},
		CallLoc2:   feedback.CallLocationInfo{File: "y.go", Line: 2},
		IsObserved: true,
	}
	cg.AddPair([]*feedback.GortPairInfo{pair})

	if len(cg.CoveredConPairs) != 1 {
		t.Errorf("CoveredConPairs len = %d, want 1", len(cg.CoveredConPairs))
	}
	if len(cg.SusConPairs) != 0 {
		t.Errorf("SusConPairs len = %d, want 0", len(cg.SusConPairs))
	}
}

func TestCorpusGort_AddPair_Suspect(t *testing.T) {
	var phase uint32 = 0
	cg := NewCorpusGort(&phase)

	pair := &feedback.GortPairInfo{
		Gid1: 1, Gid2: 2,
		CallLoc1:   feedback.CallLocationInfo{File: "x.go", Line: 1},
		CallLoc2:   feedback.CallLocationInfo{File: "y.go", Line: 2},
		IsObserved: false,
		Confidence: 0.8,
	}
	cg.AddPair([]*feedback.GortPairInfo{pair})

	if len(cg.SusConPairs) != 1 {
		t.Errorf("SusConPairs len = %d, want 1", len(cg.SusConPairs))
	}
	if len(cg.CoveredConPairs) != 0 {
		t.Errorf("CoveredConPairs len = %d, want 0", len(cg.CoveredConPairs))
	}
}

func TestCorpusGort_AddPair_SuspectDedup(t *testing.T) {
	var phase uint32 = 0
	cg := NewCorpusGort(&phase)

	locA := feedback.CallLocationInfo{File: "x.go", Line: 1}
	locB := feedback.CallLocationInfo{File: "y.go", Line: 2}

	// 首次写入：confidence=0.8
	first := &feedback.GortPairInfo{
		Gid1: 1, Gid2: 2,
		CallLoc1: locA, CallLoc2: locB,
		IsObserved: false,
		Confidence: 0.8,
	}
	cg.AddPair([]*feedback.GortPairInfo{first})

	// 同key再次写入：Gid顺序颠倒但 gortPairKey 会排序，CallLoc 与 first 一致
	second := &feedback.GortPairInfo{
		Gid1: 2, Gid2: 1,
		CallLoc1: locB, CallLoc2: locA,
		IsObserved: false,
		Confidence: 0.5,
	}
	cg.AddPair([]*feedback.GortPairInfo{second})

	if len(cg.SusConPairs) != 1 {
		t.Fatalf("SusConPairs len = %d, want 1", len(cg.SusConPairs))
	}
	// 取唯一元素
	var got *feedback.GortPairInfo
	for _, v := range cg.SusConPairs {
		got = v
		break
	}
	if got.Confidence != 0.8 {
		t.Errorf("Confidence = %.2f, want 0.80 (first write wins)", got.Confidence)
	}
}

// ---------- gortScore ----------

func TestCorpusGort_GortScore(t *testing.T) {
	var phase uint32 = 0
	cg := NewCorpusGort(&phase)

	pair := &feedback.GortPairInfo{
		Gid1: 1, Gid2: 2,
		CallLoc1:   feedback.CallLocationInfo{File: "x.go", Line: 1},
		CallLoc2:   feedback.CallLocationInfo{File: "y.go", Line: 2},
		Confidence: 0.9,
	}
	cg.AddPair([]*feedback.GortPairInfo{pair})
	key := gortPairKey(pair)

	// 无超时：score = 0.9*10 - 0*2 = 9.0
	if score := cg.gortScore(key); score != 9.0 {
		t.Errorf("gortScore() = %.2f, want 9.00", score)
	}

	// 模拟超时累加：score = 9.0 - 2 = 7.0
	cg.pairTimeouts[key] = 1
	if score := cg.gortScore(key); score != 7.0 {
		t.Errorf("gortScore() after 1 timeout = %.2f, want 7.00", score)
	}

	// 不存在的 key
	if score := cg.gortScore("nonexistent"); score != -1 {
		t.Errorf("gortScore() for nonexistent = %.2f, want -1.00", score)
	}
}

// ---------- TryEndPreExec ----------

func TestCorpusGort_TryEndPreExec_StableThreshold(t *testing.T) {
	var phase uint32 = 0
	cg := NewCorpusGort(&phase)

	pair := &feedback.GortPairInfo{
		Gid1: 1, Gid2: 2,
		CallLoc1:   feedback.CallLocationInfo{File: "a.go", Line: 10},
		CallLoc2:   feedback.CallLocationInfo{File: "b.go", Line: 20},
		IsObserved: false,
	}
	cg.AddPair([]*feedback.GortPairInfo{pair})

	// 阈值=3，但首轮因 prevPairTotal 从 0→1 会 reset stableCount，
	// 需要 4 次调用才能使 stableCount >= 3
	for i := 0; i <= gortDefaultStableThreshold; i++ {
		cg.TryEndPreExec(100)
		cg.prevPairTotal = len(cg.CoveredConPairs) + len(cg.SusConPairs)
	}

	if atomic.LoadUint32(&phase) != 1 {
		t.Errorf("GortPhase should be 1 after stable rounds, got %d", phase)
	}
	if len(cg.TryPairs) == 0 {
		t.Error("TryPairs should be filled after pre-exec ends")
	}
}

func TestCorpusGort_TryEndPreExec_MaxRounds(t *testing.T) {
	var phase uint32 = 0
	cg := NewCorpusGort(&phase)

	pair := &feedback.GortPairInfo{
		Gid1: 1, Gid2: 2,
		CallLoc1:   feedback.CallLocationInfo{File: "a.go", Line: 10},
		CallLoc2:   feedback.CallLocationInfo{File: "b.go", Line: 20},
		IsObserved: false,
	}
	cg.AddPair([]*feedback.GortPairInfo{pair})

	// 每次 total 不同（不触发 stable），但轮次达到 maxRounds
	for i := 0; i < 5; i++ {
		cg.TryEndPreExec(5)
		// 不设置 prevPairTotal 保持与当前一致，让 stableCount 每轮都 reset
		cg.prevPairTotal = 0 // 确保 total != prevPairTotal，会 reset stableCount
	}

	if atomic.LoadUint32(&phase) != 1 {
		t.Errorf("GortPhase should be 1 after maxRounds, got %d", phase)
	}
}

// ---------- RefillTryPairs ----------

func TestCorpusGort_RefillTryPairs(t *testing.T) {
	var phase uint32 = 0
	cg := NewCorpusGort(&phase)

	// 添加 3 个 SusConPairs，confidence 不同
	for i, conf := range []float64{0.5, 0.9, 0.7} {
		cg.SusConPairs[gortSignalKey(uint64(i+1), uint64(i+2))] = &feedback.GortPairInfo{
			Gid1: uint64(i + 1), Gid2: uint64(i + 2),
			Confidence: conf,
		}
	}

	// selectNum=1 只选取最高分（confidence=0.9 的那个）
	cg.selectNum = 1
	cg.RefillTryPairs()

	if len(cg.TryPairs) != 1 {
		t.Fatalf("TryPairs len = %d, want 1", len(cg.TryPairs))
	}

	// selectNum=3 选取全部
	cg.selectNum = 3
	cg.RefillTryPairs()
	if len(cg.TryPairs) != 3 {
		t.Errorf("TryPairs len = %d, want 3", len(cg.TryPairs))
	}
}

// ---------- ApplySignals ----------

func TestCorpusGort_ApplySignals_Covered(t *testing.T) {
	var phase uint32 = 1 // fuzzing 阶段
	cg := NewCorpusGort(&phase)

	loc1 := feedback.CallLocationInfo{File: "a.go", Line: 1}
	loc2 := feedback.CallLocationInfo{File: "b.go", Line: 2}

	pair := &feedback.GortPairInfo{
		Gid1: 10, Gid2: 20,
		CallLoc1: loc1, CallLoc2: loc2,
		IsObserved: false, Confidence: 0.7,
	}
	key := gortPairKey(pair)
	cg.SusConPairs[key] = pair
	cg.TryPairs[key] = pair

	signals := []*feedback.CoverageSignal{
		{PreID: 10, NextID: 20, Success: true, Kind: feedback.SignalGortCovered},
	}
	newly := cg.ApplySignals(signals)

	// 应从 SusConPairs 移入 CoveredConPairs
	if _, ok := cg.SusConPairs[key]; ok {
		t.Error("pair should be removed from SusConPairs after COVERED")
	}
	if _, ok := cg.CoveredConPairs[key]; !ok {
		t.Error("pair should be in CoveredConPairs after COVERED")
	}
	if len(newly) != 1 {
		t.Errorf("newlyCovered len = %d, want 1", len(newly))
	}
}

func TestCorpusGort_ApplySignals_TimeoutRemoval(t *testing.T) {
	var phase uint32 = 1
	cg := NewCorpusGort(&phase)

	loc1 := feedback.CallLocationInfo{File: "a.go", Line: 1}
	loc2 := feedback.CallLocationInfo{File: "b.go", Line: 2}

	pair := &feedback.GortPairInfo{
		Gid1: 10, Gid2: 20,
		CallLoc1: loc1, CallLoc2: loc2,
	}
	key := gortPairKey(pair)
	cg.SusConPairs[key] = pair
	cg.TryPairs[key] = pair

	// 累积 gortMaxTimeouts 次超时
	signal := &feedback.CoverageSignal{
		PreID: 10, NextID: 20, Success: false, Kind: feedback.SignalGortTimeout,
	}
	for i := 0; i < gortMaxTimeouts; i++ {
		cg.ApplySignals([]*feedback.CoverageSignal{signal})
		// 恢复 TryPairs，因为 RefillTryPairs 会清空
		if _, ok := cg.SusConPairs[key]; ok {
			cg.TryPairs[key] = pair
		}
	}

	if _, ok := cg.SusConPairs[key]; ok {
		t.Error("pair should be removed from SusConPairs after max timeouts")
	}
	if _, ok := cg.InfeasiblePairs[key]; !ok {
		t.Error("pair should be in InfeasiblePairs after max timeouts")
	}
}

func TestCorpusGort_ApplySignals_SelectNumExpansion(t *testing.T) {
	var phase uint32 = 1
	cg := NewCorpusGort(&phase)

	loc1 := feedback.CallLocationInfo{File: "a.go", Line: 1}
	loc2 := feedback.CallLocationInfo{File: "b.go", Line: 2}

	pair := &feedback.GortPairInfo{
		Gid1: 10, Gid2: 20,
		CallLoc1: loc1, CallLoc2: loc2,
	}
	key := gortPairKey(pair)
	cg.SusConPairs[key] = pair
	cg.TryPairs[key] = pair
	origSelectNum := cg.selectNum

	// TIMEOUT 信号，无 COVERED → selectNum 翻倍
	cg.ApplySignals([]*feedback.CoverageSignal{
		{PreID: 10, NextID: 20, Success: false, Kind: feedback.SignalGortTimeout},
	})
	if cg.selectNum != origSelectNum*2 {
		t.Errorf("selectNum = %d, want %d (doubled)", cg.selectNum, origSelectNum*2)
	}
}

// ---------- Get 阶段判断 ----------

func TestCorpusGort_Get_PreExecReturnsNil(t *testing.T) {
	var phase uint32 = 0 // 预执行阶段
	cg := NewCorpusGort(&phase)

	loc1 := feedback.CallLocationInfo{File: "a.go", Line: 1}
	loc2 := feedback.CallLocationInfo{File: "b.go", Line: 2}

	pair := &feedback.GortPairInfo{
		Gid1: 10, Gid2: 20,
		CallLoc1: loc1, CallLoc2: loc2,
	}
	cg.TryPairs[gortPairKey(pair)] = pair

	if out := cg.Get(); out != nil {
		t.Error("Get() should return nil during pre-exec phase")
	}
}
