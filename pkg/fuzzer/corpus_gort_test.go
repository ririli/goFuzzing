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
	key := feedback.GortPairKey(pair)
	// 50-100，callLoc 对应 gid 顺序也要交换
	want := "50-100|b.go:20-a.go:10"
	if key != want {
		t.Errorf("feedback.GortPairKey() = %q, want %q", key, want)
	}

	// 已有序时不变
	pair2 := &feedback.GortPairInfo{
		Gid1: 10, Gid2: 20,
		CallLoc1: loc1, CallLoc2: loc2,
	}
	key2 := feedback.GortPairKey(pair2)
	want2 := "10-20|a.go:10-b.go:20"
	if key2 != want2 {
		t.Errorf("feedback.GortPairKey() = %q, want %q", key2, want2)
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
	if cg.parents == nil || cg.children == nil {
		t.Error("topology maps should be initialized")
	}
	if cg.edgeHits == nil || cg.edgeRuns == nil {
		t.Error("edge statistics should be initialized")
	}
	if cg.selectNum != 1 {
		t.Errorf("selectNum = %d, want 1", cg.selectNum)
	}
	if cg.phase != &phase {
		t.Error("phase should point to phase")
	}
}

// ---------- AddGortEdges ----------

func TestCorpusGort_AddGortEdges_MultiRunUnion(t *testing.T) {
	var phase uint32
	cg := NewCorpusGort(&phase)

	added := cg.AddGortEdges([]*feedback.GortEdge{
		{ParentGid: 0, ChildGid: 10, Count: 2},
		{ParentGid: 0, ChildGid: 10, Count: 3},
		{ParentGid: 20, ChildGid: 30, Count: 1},
		{ParentGid: 60, ChildGid: 70, Count: 0},
		{ParentGid: 40, ChildGid: 40, Count: 1},
		{ParentGid: 50, ChildGid: 0, Count: 1},
		nil,
	})
	if added != 2 {
		t.Fatalf("AddGortEdges() added = %d, want 2", added)
	}

	topLevel := gortEdgeKey{parent: 0, child: 10}
	if cg.edgeHits[topLevel] != 5 || cg.edgeRuns[topLevel] != 1 {
		t.Fatalf("top-level edge stats = (%d hits, %d runs), want (5, 1)",
			cg.edgeHits[topLevel], cg.edgeRuns[topLevel])
	}
	if _, ok := cg.parents[10][0]; !ok {
		t.Error("parent 0 should be retained in topology")
	}

	added = cg.AddGortEdges([]*feedback.GortEdge{
		{ParentGid: 0, ChildGid: 10, Count: 4},
		{ParentGid: 99, ChildGid: 10, Count: 2},
	})
	if added != 1 {
		t.Fatalf("second AddGortEdges() added = %d, want 1", added)
	}
	if cg.edgeHits[topLevel] != 9 || cg.edgeRuns[topLevel] != 2 {
		t.Errorf("merged edge stats = (%d hits, %d runs), want (9, 2)",
			cg.edgeHits[topLevel], cg.edgeRuns[topLevel])
	}
	if len(cg.parents[10]) != 2 {
		t.Errorf("child 10 parent count = %d, want 2", len(cg.parents[10]))
	}
	if _, ok := cg.children[99][10]; !ok {
		t.Error("reverse child index should include 99 -> 10")
	}
	if _, ok := cg.children[60]; ok {
		t.Error("zero-count edge should be ignored")
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

func TestCorpusGort_AddPair_PhaseOneRefills(t *testing.T) {
	var phase uint32 = 1
	cg := NewCorpusGort(&phase)
	pair := &feedback.GortPairInfo{Gid1: 10, Gid2: 20, Confidence: 0.7}

	cg.AddPair([]*feedback.GortPairInfo{pair})

	if _, ok := cg.TryPairs[feedback.GortPairKey(pair)]; !ok {
		t.Error("new fuzz-stage suspect should be immediately selectable")
	}
}

func TestCorpusGort_AddPair_EnforcesGidPairStateExclusivity(t *testing.T) {
	var phase uint32
	cg := NewCorpusGort(&phase)

	suspect := &feedback.GortPairInfo{
		Gid1: 10, Gid2: 20,
		CallLoc1:   feedback.CallLocationInfo{File: "suspect.go", Line: 1},
		Confidence: 0.5,
	}
	suspectKey := feedback.GortPairKey(suspect)
	cg.AddPair([]*feedback.GortPairInfo{suspect})
	cg.pairTimeouts[suspectKey] = 2
	cg.TryPairs[suspectKey] = suspect

	observed := &feedback.GortPairInfo{
		Gid1: 20, Gid2: 10,
		CallLoc1:   feedback.CallLocationInfo{File: "observed.go", Line: 2},
		Confidence: 1,
		IsObserved: true,
	}
	cg.AddPair([]*feedback.GortPairInfo{observed})

	if countGortSignalPairs(cg.CoveredConPairs, 10, 20) != 1 {
		t.Error("observed pair should be stored exactly once as covered")
	}
	if countGortSignalPairs(cg.SusConPairs, 10, 20) != 0 {
		t.Error("observed pair should remove the same gid pair from suspects")
	}
	if len(cg.pairTimeouts) != 0 || len(cg.TryPairs) != 0 {
		t.Error("observed pair should clear stale timeout and try state")
	}

	cg.AddPair([]*feedback.GortPairInfo{
		{Gid1: 10, Gid2: 20, Confidence: 0.9},
		{Gid1: 30, Gid2: 30, Confidence: 0.9},
		{Gid1: 0, Gid2: 40, Confidence: 0.9},
	})
	if len(cg.SusConPairs) != 0 {
		t.Error("covered, self, and main-goroutine pairs must not be added as suspects")
	}
}

func TestCorpusGort_AddPair_PhaseOneObservedInfersFromTopology(t *testing.T) {
	var phase uint32 = 1
	cg := NewCorpusGort(&phase)
	cg.AddGortEdges([]*feedback.GortEdge{
		{ParentGid: 1, ChildGid: 10, Count: 1},
		{ParentGid: 1, ChildGid: 11, Count: 1},
	})

	cg.AddPair([]*feedback.GortPairInfo{{
		Gid1: 10, Gid2: 20, Confidence: 1, IsObserved: true,
	}})

	assertGortCandidate(t, cg, 11, 20, 0.5, "fuzz_inferred_sibling")
	if len(cg.TryPairs) == 0 {
		t.Error("late observed pair should refill inferred fuzz candidates")
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
	key := feedback.GortPairKey(pair)

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
		t.Errorf("Phase should be 1 after stable rounds, got %d", phase)
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
		t.Errorf("Phase should be 1 after maxRounds, got %d", phase)
	}
}

func TestCorpusGort_TryEndPreExec_TopologyGrowthResetsStability(t *testing.T) {
	var phase uint32
	cg := NewCorpusGort(&phase)

	cg.AddGortEdges([]*feedback.GortEdge{{ParentGid: 0, ChildGid: 10, Count: 1}})
	cg.TryEndPreExec(100) // new total
	cg.TryEndPreExec(100) // stable once
	cg.AddGortEdges([]*feedback.GortEdge{{ParentGid: 10, ChildGid: 20, Count: 1}})
	cg.TryEndPreExec(100) // topology growth must reset stability

	if atomic.LoadUint32(&phase) != 0 {
		t.Error("pre-exec ended before the enlarged topology stabilized")
	}
	if cg.stableCount != 0 {
		t.Errorf("stableCount = %d after a new edge, want 0", cg.stableCount)
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
	key := feedback.GortPairKey(pair)
	cg.SusConPairs[key] = pair
	cg.TryPairs[key] = pair

	signals := []*feedback.CoverageSignal{
		{PreID: 10, NextID: 20, Success: true, Kind: feedback.SignalPairCovered},
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
	if !pair.IsObserved || pair.Confidence != 1.0 || pair.SourceType != "fuzz_verified" {
		t.Errorf("verified pair metadata = observed:%v confidence:%.1f source:%q",
			pair.IsObserved, pair.Confidence, pair.SourceType)
	}
}

func TestCorpusGort_ApplySignals_InfersFromCachedTopology(t *testing.T) {
	var phase uint32 = 1
	cg := NewCorpusGort(&phase)
	cg.AddGortEdges([]*feedback.GortEdge{
		{ParentGid: 1, ChildGid: 10, Count: 1},
		{ParentGid: 1, ChildGid: 12, Count: 1},
		{ParentGid: 10, ChildGid: 11, Count: 1},
		{ParentGid: 2, ChildGid: 20, Count: 1},
	})

	anchor := &feedback.GortPairInfo{Gid1: 10, Gid2: 20, Confidence: 0.6}
	anchorKey := feedback.GortPairKey(anchor)
	cg.SusConPairs[anchorKey] = anchor
	cg.TryPairs[anchorKey] = anchor
	lowConfidence := &feedback.GortPairInfo{
		Gid1: 1, Gid2: 20,
		CallLoc1:   feedback.CallLocationInfo{File: "known.go", Line: 7},
		Confidence: 0.1,
		SourceType: "old_inference",
	}
	cg.SusConPairs[feedback.GortPairKey(lowConfidence)] = lowConfidence

	newly := cg.ApplySignals([]*feedback.CoverageSignal{{
		PreID: 10, NextID: 20, Success: true, Kind: feedback.SignalPairCovered,
	}})
	if len(newly) != 1 || newly[0] != anchor {
		t.Fatalf("ApplySignals() newly covered = %v, want anchor", newly)
	}
	if lowConfidence.Confidence != 0.5 || lowConfidence.SourceType != "fuzz_inferred_adjacent" {
		t.Errorf("existing parent candidate was not upgraded: confidence=%.1f source=%q",
			lowConfidence.Confidence, lowConfidence.SourceType)
	}

	assertGortCandidate(t, cg, 11, 20, 0.3, "fuzz_inferred_adjacent")
	assertGortCandidate(t, cg, 12, 20, 0.5, "fuzz_inferred_sibling")
	assertGortCandidate(t, cg, 2, 10, 0.5, "fuzz_inferred_adjacent")
	if len(cg.TryPairs) == 0 {
		t.Error("inferred candidates should refill TryPairs")
	}
}

func TestCorpusGort_ApplySignals_FiltersKnownGidPairs(t *testing.T) {
	var phase uint32 = 1
	cg := NewCorpusGort(&phase)
	cg.AddGortEdges([]*feedback.GortEdge{
		{ParentGid: 1, ChildGid: 10, Count: 1},
		{ParentGid: 10, ChildGid: 11, Count: 1},
		{ParentGid: 1, ChildGid: 12, Count: 1},
	})

	anchor := &feedback.GortPairInfo{Gid1: 10, Gid2: 20, Confidence: 0.6}
	anchorKey := feedback.GortPairKey(anchor)
	cg.SusConPairs[anchorKey] = anchor
	cg.TryPairs[anchorKey] = anchor
	covered := &feedback.GortPairInfo{
		Gid1: 20, Gid2: 1,
		CallLoc1: feedback.CallLocationInfo{File: "different.go", Line: 1},
	}
	infeasible := &feedback.GortPairInfo{
		Gid1: 20, Gid2: 11,
		CallLoc1: feedback.CallLocationInfo{File: "different.go", Line: 2},
	}
	cg.CoveredConPairs[feedback.GortPairKey(covered)] = covered
	cg.InfeasiblePairs[feedback.GortPairKey(infeasible)] = infeasible

	cg.ApplySignals([]*feedback.CoverageSignal{{
		PreID: 10, NextID: 20, Success: true, Kind: feedback.SignalPairCovered,
	}})

	if countGortSignalPairs(cg.SusConPairs, 1, 20) != 0 {
		t.Error("covered unordered gid pair should block inferred duplicate")
	}
	if countGortSignalPairs(cg.SusConPairs, 11, 20) != 0 {
		t.Error("infeasible unordered gid pair should block inferred duplicate")
	}
	if countGortSignalPairs(cg.SusConPairs, 12, 20) != 1 {
		t.Error("unknown sibling pair should still be inferred")
	}
}

func TestCorpusGort_AddGortEdges_PhaseOneInfersFromAllCovered(t *testing.T) {
	var phase uint32 = 1
	cg := NewCorpusGort(&phase)
	anchor := &feedback.GortPairInfo{
		Gid1: 10, Gid2: 20, Confidence: 1, IsObserved: true,
	}
	cg.CoveredConPairs[feedback.GortPairKey(anchor)] = anchor

	added := cg.AddGortEdges([]*feedback.GortEdge{
		{ParentGid: 0, ChildGid: 10, Count: 1},
		{ParentGid: 0, ChildGid: 12, Count: 1},
	})

	if added != 2 {
		t.Fatalf("AddGortEdges() added = %d, want 2", added)
	}
	assertGortCandidate(t, cg, 12, 20, 0.5, "fuzz_inferred_sibling")
	if len(cg.TryPairs) != 1 {
		t.Errorf("late topology should refill TryPairs, got %d entries", len(cg.TryPairs))
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
	key := feedback.GortPairKey(pair)
	cg.SusConPairs[key] = pair
	cg.TryPairs[key] = pair

	// 累积 gortMaxTimeouts 次超时
	signal := &feedback.CoverageSignal{
		PreID: 10, NextID: 20, Success: false, Kind: feedback.SignalPairTimeout,
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
	key := feedback.GortPairKey(pair)
	cg.SusConPairs[key] = pair
	cg.TryPairs[key] = pair
	origSelectNum := cg.selectNum

	// TIMEOUT 信号，无 COVERED → selectNum 翻倍
	cg.ApplySignals([]*feedback.CoverageSignal{
		{PreID: 10, NextID: 20, Success: false, Kind: feedback.SignalPairTimeout},
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
	cg.TryPairs[feedback.GortPairKey(pair)] = pair

	if out := cg.Get(); out != nil {
		t.Error("Get() should return nil during pre-exec phase")
	}
}

func assertGortCandidate(t *testing.T, cg *CorpusGort, gid1, gid2 uint64, confidence float64, source string) {
	t.Helper()
	wantKey := gortSignalKey(gid1, gid2)
	for _, pair := range cg.SusConPairs {
		if pair == nil || gortSignalKey(pair.Gid1, pair.Gid2) != wantKey {
			continue
		}
		if pair.Confidence != confidence || pair.SourceType != source || pair.IsObserved {
			t.Errorf("candidate %s = confidence %.1f, source %q, observed %v; want %.1f, %q, false",
				wantKey, pair.Confidence, pair.SourceType, pair.IsObserved, confidence, source)
		}
		return
	}
	t.Errorf("candidate %s not found", wantKey)
}

func countGortSignalPairs(pairs map[string]*feedback.GortPairInfo, gid1, gid2 uint64) int {
	wantKey := gortSignalKey(gid1, gid2)
	count := 0
	for _, pair := range pairs {
		if pair != nil && gortSignalKey(pair.Gid1, pair.Gid2) == wantKey {
			count++
		}
	}
	return count
}
