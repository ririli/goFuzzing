package fuzzer

import (
	"sync/atomic"
	"testing"

	"toolkit/pkg/feedback"
)

// ---------- 编译期接口断言 ----------

var (
	_ PairCorpus = (*CorpusGort)(nil)
	_ PairCorpus = (*CorpusFunc)(nil)
)

// ---------- 键生成 ----------

func TestFuncPairKey(t *testing.T) {
	loc1 := feedback.CallLocationInfo{File: "a.go", Line: 10}
	loc2 := feedback.CallLocationInfo{File: "b.go", Line: 20}

	// 小的 FuncID 应在前面，CallLoc 随 ID 顺序交换
	pair := &feedback.SuspiciousPairInfo{
		FuncID1: 100, FuncID2: 50,
		CallLoc1: loc1, CallLoc2: loc2,
	}
	if key, want := pair.PairKey(), "50-100|b.go:20-a.go:10"; key != want {
		t.Errorf("PairKey() = %q, want %q", key, want)
	}

	pair2 := &feedback.SuspiciousPairInfo{
		FuncID1: 10, FuncID2: 20,
		CallLoc1: loc1, CallLoc2: loc2,
	}
	if key, want := pair2.PairKey(), "10-20|a.go:10-b.go:20"; key != want {
		t.Errorf("PairKey() = %q, want %q", key, want)
	}
}

func TestFuncSignalKey(t *testing.T) {
	// SignalKey 不含调用位置，且方向无关
	pair := &feedback.SuspiciousPairInfo{
		FuncID1: 100, FuncID2: 50,
		CallLoc1: feedback.CallLocationInfo{File: "a.go", Line: 1},
	}
	if got, want := pair.SignalKey(), "50-100"; got != want {
		t.Errorf("SignalKey() = %q, want %q", got, want)
	}
	if got := funcSignalKey(10, 20); got != "10-20" {
		t.Errorf("funcSignalKey(10,20) = %q, want '10-20'", got)
	}
	if got := pairSignalKey(20, 10); got != "10-20" {
		t.Errorf("pairSignalKey(20,10) = %q, want '10-20'", got)
	}
}

// ---------- 构造 ----------

func TestNewCorpusFunc(t *testing.T) {
	var phase uint32
	cf := NewCorpusFunc(&phase)

	if cf.CoveredConPairs == nil {
		t.Error("CoveredConPairs should be initialized")
	}
	if cf.SusConPairs == nil {
		t.Error("SusConPairs should be initialized")
	}
	if cf.InfeasiblePairs == nil {
		t.Error("InfeasiblePairs should be initialized")
	}
	if cf.TryPairs == nil {
		t.Error("TryPairs should be initialized")
	}
	if cf.pairTimeouts == nil {
		t.Error("pairTimeouts should be initialized")
	}
	if cf.callers == nil || cf.callees == nil {
		t.Error("topology maps should be initialized")
	}
	if cf.edgeHits == nil || cf.edgeRuns == nil {
		t.Error("edge statistics should be initialized")
	}
	if cf.selectNum != 1 {
		t.Errorf("selectNum = %d, want 1", cf.selectNum)
	}
	if cf.phase != &phase {
		t.Error("phase should point to phase")
	}
	if cf.ModeName() != "function" {
		t.Errorf("ModeName() = %q, want 'function'", cf.ModeName())
	}
}

// ---------- AddFuncEdges ----------

func TestCorpusFunc_AddFuncEdges_MultiRunUnion(t *testing.T) {
	var phase uint32
	cf := NewCorpusFunc(&phase)

	added := cf.AddFuncEdges([]*feedback.FuncEdge{
		{Caller: 1, Callee: 10, Count: 2},
		{Caller: 1, Callee: 10, Count: 3},
		{Caller: 2, Callee: 30, Count: 1},
		{Caller: 6, Callee: 7, Count: 0}, // Count=0 忽略
		{Caller: 4, Callee: 4, Count: 1}, // 自环忽略
		{Caller: 5, Callee: 0, Count: 1}, // callee=0 忽略
		nil,
	})
	if added != 2 {
		t.Fatalf("AddFuncEdges() added = %d, want 2", added)
	}

	topEdge := funcEdgeKey{caller: 1, callee: 10}
	if cf.edgeHits[topEdge] != 5 || cf.edgeRuns[topEdge] != 1 {
		t.Fatalf("edge stats = (%d hits, %d runs), want (5, 1)",
			cf.edgeHits[topEdge], cf.edgeRuns[topEdge])
	}
	if _, ok := cf.callers[10][1]; !ok {
		t.Error("caller 1 should be retained in topology")
	}

	added = cf.AddFuncEdges([]*feedback.FuncEdge{
		{Caller: 1, Callee: 10, Count: 4},
		{Caller: 9, Callee: 10, Count: 2},
	})
	if added != 1 {
		t.Fatalf("second AddFuncEdges() added = %d, want 1", added)
	}
	if cf.edgeHits[topEdge] != 9 || cf.edgeRuns[topEdge] != 2 {
		t.Errorf("merged edge stats = (%d hits, %d runs), want (9, 2)",
			cf.edgeHits[topEdge], cf.edgeRuns[topEdge])
	}
	if len(cf.callers[10]) != 2 {
		t.Errorf("callee 10 caller count = %d, want 2", len(cf.callers[10]))
	}
	if _, ok := cf.callees[9][10]; !ok {
		t.Error("reverse callee index should include 9 -> 10")
	}
	if _, ok := cf.callees[6]; ok {
		t.Error("zero-count edge should be ignored")
	}
}

// ---------- AddPair ----------

func TestCorpusFunc_AddPair_Observed(t *testing.T) {
	var phase uint32 = 0
	cf := NewCorpusFunc(&phase)

	pair := &feedback.SuspiciousPairInfo{
		FuncID1: 1, FuncID2: 2,
		CallLoc1:   feedback.CallLocationInfo{File: "x.go", Line: 1},
		CallLoc2:   feedback.CallLocationInfo{File: "y.go", Line: 2},
		IsObserved: true,
	}
	cf.AddPair([]*feedback.SuspiciousPairInfo{pair})

	if len(cf.CoveredConPairs) != 1 {
		t.Errorf("CoveredConPairs len = %d, want 1", len(cf.CoveredConPairs))
	}
	if len(cf.SusConPairs) != 0 {
		t.Errorf("SusConPairs len = %d, want 0", len(cf.SusConPairs))
	}
}

func TestCorpusFunc_AddPair_SuspectDedup(t *testing.T) {
	var phase uint32 = 0
	cf := NewCorpusFunc(&phase)

	locA := feedback.CallLocationInfo{File: "x.go", Line: 1}
	locB := feedback.CallLocationInfo{File: "y.go", Line: 2}

	first := &feedback.SuspiciousPairInfo{
		FuncID1: 1, FuncID2: 2,
		CallLoc1: locA, CallLoc2: locB,
		Confidence: 0.8,
	}
	cf.AddPair([]*feedback.SuspiciousPairInfo{first})

	// 同 signalKey 再次写入（ID 顺序颠倒），只升级置信度不新增
	second := &feedback.SuspiciousPairInfo{
		FuncID1: 2, FuncID2: 1,
		CallLoc1: locB, CallLoc2: locA,
		Confidence: 0.9,
	}
	cf.AddPair([]*feedback.SuspiciousPairInfo{second})

	if len(cf.SusConPairs) != 1 {
		t.Fatalf("SusConPairs len = %d, want 1", len(cf.SusConPairs))
	}
	for _, v := range cf.SusConPairs {
		if v.Confidence != 0.9 {
			t.Errorf("Confidence = %.2f, want 0.90 (higher confidence wins)", v.Confidence)
		}
	}
}

func TestCorpusFunc_AddPair_PhaseOneRefills(t *testing.T) {
	var phase uint32 = 1
	cf := NewCorpusFunc(&phase)
	pair := &feedback.SuspiciousPairInfo{FuncID1: 10, FuncID2: 20, Confidence: 0.7}

	cf.AddPair([]*feedback.SuspiciousPairInfo{pair})

	if _, ok := cf.TryPairs[pair.PairKey()]; !ok {
		t.Error("new fuzz-stage suspect should be immediately selectable")
	}
}

func TestCorpusFunc_AddPair_EnforcesStateExclusivity(t *testing.T) {
	var phase uint32
	cf := NewCorpusFunc(&phase)

	suspect := &feedback.SuspiciousPairInfo{
		FuncID1: 10, FuncID2: 20,
		CallLoc1:   feedback.CallLocationInfo{File: "suspect.go", Line: 1},
		Confidence: 0.5,
	}
	suspectKey := suspect.PairKey()
	cf.AddPair([]*feedback.SuspiciousPairInfo{suspect})
	cf.pairTimeouts[suspectKey] = 2
	cf.TryPairs[suspectKey] = suspect

	observed := &feedback.SuspiciousPairInfo{
		FuncID1: 20, FuncID2: 10,
		CallLoc1:   feedback.CallLocationInfo{File: "observed.go", Line: 2},
		Confidence: 1,
		IsObserved: true,
	}
	cf.AddPair([]*feedback.SuspiciousPairInfo{observed})

	if countFuncSignalPairs(cf.CoveredConPairs, 10, 20) != 1 {
		t.Error("observed pair should be stored exactly once as covered")
	}
	if countFuncSignalPairs(cf.SusConPairs, 10, 20) != 0 {
		t.Error("observed pair should remove the same func pair from suspects")
	}
	if len(cf.pairTimeouts) != 0 || len(cf.TryPairs) != 0 {
		t.Error("observed pair should clear stale timeout and try state")
	}

	cf.AddPair([]*feedback.SuspiciousPairInfo{
		{FuncID1: 10, FuncID2: 20, Confidence: 0.9},
		{FuncID1: 30, FuncID2: 30, Confidence: 0.9},
		{FuncID1: 0, FuncID2: 40, Confidence: 0.9},
	})
	if len(cf.SusConPairs) != 0 {
		t.Error("covered, self, and zero-id pairs must not be added as suspects")
	}
}

// ---------- funcScore ----------

func TestCorpusFunc_FuncScore(t *testing.T) {
	var phase uint32 = 0
	cf := NewCorpusFunc(&phase)

	pair := &feedback.SuspiciousPairInfo{
		FuncID1: 1, FuncID2: 2,
		Confidence: 0.9,
	}
	cf.AddPair([]*feedback.SuspiciousPairInfo{pair})
	key := pair.PairKey()

	if score := cf.funcScore(key); score != 9.0 {
		t.Errorf("funcScore() = %.2f, want 9.00", score)
	}
	cf.pairTimeouts[key] = 1
	if score := cf.funcScore(key); score != 7.0 {
		t.Errorf("funcScore() after 1 timeout = %.2f, want 7.00", score)
	}
	if score := cf.funcScore("nonexistent"); score != -1 {
		t.Errorf("funcScore() for nonexistent = %.2f, want -1.00", score)
	}
}

// ---------- TryEndPreExec ----------

func TestCorpusFunc_TryEndPreExec_StableThreshold(t *testing.T) {
	var phase uint32 = 0
	cf := NewCorpusFunc(&phase)

	pair := &feedback.SuspiciousPairInfo{
		FuncID1: 1, FuncID2: 2,
		CallLoc1: feedback.CallLocationInfo{File: "a.go", Line: 10},
		CallLoc2: feedback.CallLocationInfo{File: "b.go", Line: 20},
	}
	cf.AddPair([]*feedback.SuspiciousPairInfo{pair})

	// 阈值=5，首轮因 total 变化 reset，需要 threshold+1 次调用
	for i := 0; i <= funcDefaultStableThreshold; i++ {
		cf.TryEndPreExec(100)
		cf.prevPairTotal = len(cf.CoveredConPairs) + len(cf.SusConPairs)
	}

	if atomic.LoadUint32(&phase) != 1 {
		t.Errorf("Phase should be 1 after stable rounds, got %d", phase)
	}
	if len(cf.TryPairs) == 0 {
		t.Error("TryPairs should be filled after pre-exec ends")
	}
	if cf.InPreExec() {
		t.Error("InPreExec() should be false after phase transition")
	}
}

func TestCorpusFunc_TryEndPreExec_MaxRounds(t *testing.T) {
	var phase uint32 = 0
	cf := NewCorpusFunc(&phase)

	pair := &feedback.SuspiciousPairInfo{FuncID1: 1, FuncID2: 2}
	cf.AddPair([]*feedback.SuspiciousPairInfo{pair})

	for i := 0; i < 5; i++ {
		cf.TryEndPreExec(5)
		cf.prevPairTotal = 0 // 每轮 reset stableCount，仅靠 maxRounds 触发
	}

	if atomic.LoadUint32(&phase) != 1 {
		t.Errorf("Phase should be 1 after maxRounds, got %d", phase)
	}
}

func TestCorpusFunc_TryEndPreExec_TopologyGrowthResetsStability(t *testing.T) {
	var phase uint32
	cf := NewCorpusFunc(&phase)

	cf.AddFuncEdges([]*feedback.FuncEdge{{Caller: 1, Callee: 10, Count: 1}})
	cf.TryEndPreExec(100) // new total
	cf.TryEndPreExec(100) // stable once
	cf.AddFuncEdges([]*feedback.FuncEdge{{Caller: 10, Callee: 20, Count: 1}})
	cf.TryEndPreExec(100) // topology growth must reset stability

	if atomic.LoadUint32(&phase) != 0 {
		t.Error("pre-exec ended before the enlarged topology stabilized")
	}
	if cf.stableCount != 0 {
		t.Errorf("stableCount = %d after a new edge, want 0", cf.stableCount)
	}
}

// ---------- RefillTryPairs ----------

func TestCorpusFunc_RefillTryPairs(t *testing.T) {
	var phase uint32 = 0
	cf := NewCorpusFunc(&phase)

	for i, conf := range []float64{0.5, 0.9, 0.7} {
		pair := &feedback.SuspiciousPairInfo{
			FuncID1: uint64(i + 1), FuncID2: uint64(i + 2),
			Confidence: conf,
		}
		cf.SusConPairs[pair.PairKey()] = pair
	}

	cf.selectNum = 1
	cf.RefillTryPairs()
	if len(cf.TryPairs) != 1 {
		t.Fatalf("TryPairs len = %d, want 1", len(cf.TryPairs))
	}

	cf.selectNum = 3
	cf.RefillTryPairs()
	if len(cf.TryPairs) != 3 {
		t.Errorf("TryPairs len = %d, want 3", len(cf.TryPairs))
	}
}

// ---------- ApplySignals ----------

func TestCorpusFunc_ApplySignals_Covered(t *testing.T) {
	var phase uint32 = 1
	cf := NewCorpusFunc(&phase)

	pair := &feedback.SuspiciousPairInfo{
		FuncID1: 10, FuncID2: 20,
		CallLoc1:   feedback.CallLocationInfo{File: "a.go", Line: 1},
		CallLoc2:   feedback.CallLocationInfo{File: "b.go", Line: 2},
		Confidence: 0.7,
	}
	key := pair.PairKey()
	cf.SusConPairs[key] = pair
	cf.TryPairs[key] = pair

	// 函数模式复用 SignalPairCovered/SignalPairTimeout（{COVERED}/{TIMEOUT} 前缀）
	newly := cf.ApplySignals([]*feedback.CoverageSignal{
		{PreID: 10, NextID: 20, Success: true, Kind: feedback.SignalPairCovered},
	})

	if _, ok := cf.SusConPairs[key]; ok {
		t.Error("pair should be removed from SusConPairs after COVERED")
	}
	if _, ok := cf.CoveredConPairs[key]; !ok {
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

func TestCorpusFunc_ApplySignals_InfersFromCachedTopology(t *testing.T) {
	var phase uint32 = 1
	cf := NewCorpusFunc(&phase)
	cf.AddFuncEdges([]*feedback.FuncEdge{
		{Caller: 1, Callee: 10, Count: 1},
		{Caller: 1, Callee: 12, Count: 1},
		{Caller: 10, Callee: 11, Count: 1},
		{Caller: 2, Callee: 20, Count: 1},
	})

	anchor := &feedback.SuspiciousPairInfo{FuncID1: 10, FuncID2: 20, Confidence: 0.6}
	anchorKey := anchor.PairKey()
	cf.SusConPairs[anchorKey] = anchor
	cf.TryPairs[anchorKey] = anchor
	lowConfidence := &feedback.SuspiciousPairInfo{
		FuncID1: 1, FuncID2: 20,
		CallLoc1:   feedback.CallLocationInfo{File: "known.go", Line: 7},
		Confidence: 0.1,
		SourceType: "old_inference",
	}
	cf.SusConPairs[lowConfidence.PairKey()] = lowConfidence

	newly := cf.ApplySignals([]*feedback.CoverageSignal{{
		PreID: 10, NextID: 20, Success: true, Kind: feedback.SignalPairCovered,
	}})
	if len(newly) != 1 || newly[0] != anchor {
		t.Fatalf("ApplySignals() newly covered = %v, want anchor", newly)
	}
	if lowConfidence.Confidence != 0.5 || lowConfidence.SourceType != "fuzz_inferred_adjacent" {
		t.Errorf("existing caller candidate was not upgraded: confidence=%.1f source=%q",
			lowConfidence.Confidence, lowConfidence.SourceType)
	}

	assertFuncCandidate(t, cf, 11, 20, 0.3, "fuzz_inferred_adjacent")
	assertFuncCandidate(t, cf, 2, 10, 0.5, "fuzz_inferred_adjacent")
	if len(cf.TryPairs) == 0 {
		t.Error("inferred candidates should refill TryPairs")
	}
}

func TestCorpusFunc_ApplySignals_TimeoutRemoval(t *testing.T) {
	var phase uint32 = 1
	cf := NewCorpusFunc(&phase)

	pair := &feedback.SuspiciousPairInfo{FuncID1: 10, FuncID2: 20}
	key := pair.PairKey()
	cf.SusConPairs[key] = pair
	cf.TryPairs[key] = pair

	signal := &feedback.CoverageSignal{
		PreID: 10, NextID: 20, Success: false, Kind: feedback.SignalPairTimeout,
	}
	for i := 0; i < funcMaxTimeouts; i++ {
		cf.ApplySignals([]*feedback.CoverageSignal{signal})
		if _, ok := cf.SusConPairs[key]; ok {
			cf.TryPairs[key] = pair
		}
	}

	if _, ok := cf.SusConPairs[key]; ok {
		t.Error("pair should be removed from SusConPairs after max timeouts")
	}
	if _, ok := cf.InfeasiblePairs[key]; !ok {
		t.Error("pair should be in InfeasiblePairs after max timeouts")
	}
}

func TestCorpusFunc_ApplySignals_SelectNumExpansion(t *testing.T) {
	var phase uint32 = 1
	cf := NewCorpusFunc(&phase)

	pair := &feedback.SuspiciousPairInfo{FuncID1: 10, FuncID2: 20}
	key := pair.PairKey()
	cf.SusConPairs[key] = pair
	cf.TryPairs[key] = pair
	origSelectNum := cf.selectNum

	cf.ApplySignals([]*feedback.CoverageSignal{
		{PreID: 10, NextID: 20, Success: false, Kind: feedback.SignalPairTimeout},
	})
	if cf.selectNum != origSelectNum*2 {
		t.Errorf("selectNum = %d, want %d (doubled)", cf.selectNum, origSelectNum*2)
	}
}

// ---------- Get / GetInput 阶段判断 ----------

func TestCorpusFunc_Get_PreExecReturnsNil(t *testing.T) {
	var phase uint32 = 0
	cf := NewCorpusFunc(&phase)

	pair := &feedback.SuspiciousPairInfo{FuncID1: 10, FuncID2: 20}
	cf.TryPairs[pair.PairKey()] = pair

	if out := cf.Get(); out != nil {
		t.Error("Get() should return nil during pre-exec phase")
	}
	if out := cf.GetInput(); out != nil {
		t.Error("GetInput() should return nil during pre-exec phase")
	}
}

func TestCorpusFunc_GetInput_UnifiesPairs(t *testing.T) {
	var phase uint32 = 1
	cf := NewCorpusFunc(&phase)

	pair := &feedback.SuspiciousPairInfo{FuncID1: 10, FuncID2: 20}
	cf.TryPairs[pair.PairKey()] = pair

	in := cf.GetInput()
	if in == nil || in.IsEmpty() {
		t.Fatal("GetInput() should return unified PairInput in fuzzing phase")
	}
	if got, want := in.ToString(), "(10,20)"; got != want {
		t.Errorf("PairInput.ToString() = %q, want %q", got, want)
	}
	if _, ok := in.Pairs[0].(*feedback.SuspiciousPairInfo); !ok {
		t.Error("PairInput.Pairs should hold *SuspiciousPairInfo in function mode")
	}
}

// ---------- OnPreExecEnd / OnCoveredByOp（funcID 链路的 OP 联动） ----------

func TestCorpusFunc_OnPreExecEnd_GeneratesOpPairs(t *testing.T) {
	var phase uint32 = 1
	cf := NewCorpusFunc(&phase)
	co := NewCorpusOp(&phase)

	// fid=1 的函数内有 close，fid=2 的函数内有 send，操作同一对象 → close-before-send
	co.Add([]*feedback.OpInfo{
		{OpId: 1, ObjAddr: 0x100, OpType: feedback.OpTypeClose, ObjKind: feedback.OpKindChannel, FuncIDs: []uint64{1}},
		{OpId: 2, ObjAddr: 0x100, OpType: feedback.OpTypeSend, ObjKind: feedback.OpKindChannel, FuncIDs: []uint64{2}},
	})

	covered := &feedback.SuspiciousPairInfo{FuncID1: 1, FuncID2: 2, Confidence: 1, IsObserved: true}
	cf.CoveredConPairs[covered.PairKey()] = covered

	cf.OnPreExecEnd(co)

	if !co.generated {
		t.Fatal("function mode should mark OP corpus as generated")
	}
	if len(co.SusConPairs) != 1 {
		t.Fatalf("SusConPairs = %d, want 1 close-before-send pair", len(co.SusConPairs))
	}
	for _, p := range co.SusConPairs {
		if p.Danger != feedback.DangerCloseBeforeSend {
			t.Errorf("danger = %q, want %q", p.Danger, feedback.DangerCloseBeforeSend)
		}
	}
}

func TestCorpusFunc_OnCoveredByOp_GeneratesIncrementally(t *testing.T) {
	var phase uint32 = 1
	cf := NewCorpusFunc(&phase)
	co := NewCorpusOp(&phase)
	co.generated = true

	// 栈归属：close 在内层函数 7（栈含外层 5），send 在函数 2
	co.Add([]*feedback.OpInfo{
		{OpId: 1, ObjAddr: 0x200, OpType: feedback.OpTypeClose, ObjKind: feedback.OpKindChannel, FuncIDs: []uint64{5, 7}},
		{OpId: 2, ObjAddr: 0x200, OpType: feedback.OpTypeSend, ObjKind: feedback.OpKindChannel, FuncIDs: []uint64{2}},
	})

	// 新覆盖对 (5,2)：close 虽在 fid=7，但归属于外层函数 5 的动态范围
	cf.OnCoveredByOp(co, []feedback.ConcurrencyPair{
		&feedback.SuspiciousPairInfo{FuncID1: 5, FuncID2: 2},
	})

	if len(co.SusConPairs) != 1 {
		t.Fatalf("SusConPairs = %d, want 1 pair via stack attribution", len(co.SusConPairs))
	}
}

// ---------- 辅助函数 ----------

func assertFuncCandidate(t *testing.T, cf *CorpusFunc, fid1, fid2 uint64, confidence float64, source string) {
	t.Helper()
	wantKey := funcSignalKey(fid1, fid2)
	for _, pair := range cf.SusConPairs {
		if pair == nil || pair.SignalKey() != wantKey {
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

func countFuncSignalPairs(pairs map[string]*feedback.SuspiciousPairInfo, fid1, fid2 uint64) int {
	wantKey := funcSignalKey(fid1, fid2)
	count := 0
	for _, pair := range pairs {
		if pair != nil && pair.SignalKey() == wantKey {
			count++
		}
	}
	return count
}
