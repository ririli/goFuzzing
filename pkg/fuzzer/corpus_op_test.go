package fuzzer

import (
	"testing"

	"toolkit/pkg/feedback"
)

// ---------- 辅助函数 ----------

func makeOpInfo(opId, gid, objAddr uint64, opType feedback.OpType, objKind feedback.OpKind) *feedback.OpInfo {
	return &feedback.OpInfo{
		OpId: opId, Gid: gid, ObjAddr: objAddr,
		OpType: opType, ObjKind: objKind,
		IsSelect: false,
	}
}

// ---------- 键生成 ----------

func TestOpPairKey(t *testing.T) {
	a := makeOpInfo(1, 10, 0xABCD, feedback.OpTypeClose, feedback.OpKindChannel)
	b := makeOpInfo(2, 20, 0xABCD, feedback.OpTypeSend, feedback.OpKindChannel)

	pair := &feedback.OpPair{Op1: a, Op2: b, Danger: feedback.DangerCloseBeforeSend}
	key := opPairKey(pair)

	// Danger-ObjAddr-OpId1-OpId2
	want := "close-before-send-43981-1-2"
	if key != want {
		t.Errorf("opPairKey() = %q, want %q", key, want)
	}
}

func TestOpSignalKey(t *testing.T) {
	if got := opSignalKey(100, 200); got != "100-200" {
		t.Errorf("opSignalKey(100, 200) = %q, want '100-200'", got)
	}
	if got := opSignalKey(200, 100); got != "200-100" {
		t.Errorf("opSignalKey(200, 100) = %q, want '200-100' (direction-sensitive)", got)
	}
}

// ---------- 构造 ----------

func TestNewCorpusOp(t *testing.T) {
	var phase uint32
	co := NewCorpusOp(&phase)

	if co.ops == nil {
		t.Error("ops should be initialized")
	}
	if co.byGid == nil {
		t.Error("byGid should be initialized")
	}
	if co.CoveredConPairs == nil {
		t.Error("CoveredConPairs should be initialized")
	}
	if co.SusConPairs == nil {
		t.Error("SusConPairs should be initialized")
	}
	if co.InfeasiblePairs == nil {
		t.Error("InfeasiblePairs should be initialized")
	}
	if co.TryPairs == nil {
		t.Error("TryPairs should be initialized")
	}
	if co.pairTimeouts == nil {
		t.Error("pairTimeouts should be initialized")
	}
	if co.selectNum != 1 {
		t.Errorf("selectNum = %d, want 1", co.selectNum)
	}
	if co.generated {
		t.Error("generated should be false")
	}
	if co.gortPhase != &phase {
		t.Error("gortPhase should point to phase")
	}
}

// ---------- Add ----------

func TestCorpusOp_Add(t *testing.T) {
	co := NewCorpusOp(new(uint32))

	op := makeOpInfo(1, 10, 0x100, feedback.OpTypeSend, feedback.OpKindChannel)
	co.Add([]*feedback.OpInfo{op})

	if _, ok := co.ops[1]; !ok {
		t.Error("ops[1] should exist after Add")
	}
	if len(co.byGid[10]) != 1 {
		t.Errorf("byGid[10] len = %d, want 1", len(co.byGid[10]))
	}
	// 检查 collected
	if !co.collected {
		t.Error("collected should be true after first Add in pre-exec")
	}
}

func TestCorpusOp_Add_Dedup(t *testing.T) {
	co := NewCorpusOp(new(uint32))

	op1 := makeOpInfo(1, 10, 0x100, feedback.OpTypeSend, feedback.OpKindChannel)
	op2 := makeOpInfo(1, 10, 0x100, feedback.OpTypeSend, feedback.OpKindChannel) // same OpId
	co.Add([]*feedback.OpInfo{op1, op2})

	if len(co.ops) != 1 {
		t.Errorf("ops len = %d, want 1 (dedup by OpId)", len(co.ops))
	}
}

func TestCorpusOp_Add_SkipAfterCollected(t *testing.T) {
	var phase uint32 = 0
	co := NewCorpusOp(&phase)

	// 预执行阶段首次收集
	op := makeOpInfo(1, 10, 0x100, feedback.OpTypeSend, feedback.OpKindChannel)
	co.Add([]*feedback.OpInfo{op})

	if !co.collected {
		t.Fatal("collected should be true after first Add")
	}

	// 第二次添加应该被跳过
	op2 := makeOpInfo(2, 20, 0x200, feedback.OpTypeClose, feedback.OpKindChannel)
	co.Add([]*feedback.OpInfo{op2})

	if _, ok := co.ops[2]; ok {
		t.Error("ops[2] should NOT exist — Add skipped after collected")
	}
}

func TestCorpusOp_Add_NotSkipAfterFuzzing(t *testing.T) {
	var phase uint32 = 1 // fuzzing 阶段
	co := NewCorpusOp(&phase)

	op := makeOpInfo(1, 10, 0x100, feedback.OpTypeSend, feedback.OpKindChannel)
	co.Add([]*feedback.OpInfo{op})

	// fuzzing 阶段 collected 仍为 false
	if co.collected {
		t.Error("collected should still be false in fuzzing phase")
	}
}

// ---------- generateFromPair ----------

func TestCorpusOp_GenerateFromPair_Channel(t *testing.T) {
	co := NewCorpusOp(new(uint32))

	// Gid 10: close(ch, 0xABCD)
	// Gid 20: send(ch, 0xABCD)
	opClose := makeOpInfo(1, 10, 0xABCD, feedback.OpTypeClose, feedback.OpKindChannel)
	opSend := makeOpInfo(2, 20, 0xABCD, feedback.OpTypeSend, feedback.OpKindChannel)
	co.ops[1] = opClose
	co.ops[2] = opSend
	co.byGid[10] = map[uint64]struct{}{1: {}}
	co.byGid[20] = map[uint64]struct{}{2: {}}

	pair := &feedback.GortPairInfo{Gid1: 10, Gid2: 20}
	seen := make(map[string]struct{})
	co.generateFromPair(pair, seen)

	// close→send 和 send→close 都应尝试:
	// close(10)→send(20): MatchOpPair(close,send)=close-before-send ✓
	// send(20)→close(10): MatchOpPair(send,close)=nil ✗
	if len(co.SusConPairs) != 1 {
		t.Fatalf("SusConPairs len = %d, want 1", len(co.SusConPairs))
	}
	for _, v := range co.SusConPairs {
		if v.Danger != feedback.DangerCloseBeforeSend {
			t.Errorf("Danger = %q, want %q", v.Danger, feedback.DangerCloseBeforeSend)
		}
	}
}

func TestCorpusOp_GenerateFromPair_WaitGroup(t *testing.T) {
	co := NewCorpusOp(new(uint32))

	// Gid 10: done(wg, 0x200)
	// Gid 20: add(wg, 0x200)
	opDone := makeOpInfo(1, 10, 0x200, feedback.OpTypeDone, feedback.OpKindWaitGroup)
	opAdd := makeOpInfo(2, 20, 0x200, feedback.OpTypeAdd, feedback.OpKindWaitGroup)
	co.ops[1] = opDone
	co.ops[2] = opAdd
	co.byGid[10] = map[uint64]struct{}{1: {}}
	co.byGid[20] = map[uint64]struct{}{2: {}}

	pair := &feedback.GortPairInfo{Gid1: 10, Gid2: 20}
	seen := make(map[string]struct{})
	co.generateFromPair(pair, seen)

	if len(co.SusConPairs) != 1 {
		t.Fatalf("SusConPairs len = %d, want 1", len(co.SusConPairs))
	}
	for _, v := range co.SusConPairs {
		if v.Danger != feedback.DangerDoneBeforeAdd {
			t.Errorf("Danger = %q, want %q", v.Danger, feedback.DangerDoneBeforeAdd)
		}
	}
}

func TestCorpusOp_GenerateFromPair_NoMatch(t *testing.T) {
	co := NewCorpusOp(new(uint32))

	// 不同对象
	opSend := makeOpInfo(1, 10, 0x111, feedback.OpTypeSend, feedback.OpKindChannel)
	opClose := makeOpInfo(2, 20, 0x222, feedback.OpTypeClose, feedback.OpKindChannel)
	co.ops[1] = opSend
	co.ops[2] = opClose
	co.byGid[10] = map[uint64]struct{}{1: {}}
	co.byGid[20] = map[uint64]struct{}{2: {}}

	pair := &feedback.GortPairInfo{Gid1: 10, Gid2: 20}
	seen := make(map[string]struct{})
	co.generateFromPair(pair, seen)

	if len(co.SusConPairs) != 0 {
		t.Errorf("SusConPairs len = %d, want 0 (different objects, no match)", len(co.SusConPairs))
	}
}

func TestCorpusOp_GenerateFromPair_SelectConstraint(t *testing.T) {
	co := NewCorpusOp(new(uint32))

	// Gid 10: close(ch, 0x100)
	// Gid 20: send(ch, 0x100) but IsSelect=true → 只能做 Op1，不能做 Op2
	opClose := makeOpInfo(1, 10, 0x100, feedback.OpTypeClose, feedback.OpKindChannel)
	opSend := makeOpInfo(2, 20, 0x100, feedback.OpTypeSend, feedback.OpKindChannel)
	opSend.IsSelect = true
	co.ops[1] = opClose
	co.ops[2] = opSend
	co.byGid[10] = map[uint64]struct{}{1: {}}
	co.byGid[20] = map[uint64]struct{}{2: {}}

	pair := &feedback.GortPairInfo{Gid1: 10, Gid2: 20}
	seen := make(map[string]struct{})
	co.generateFromPair(pair, seen)

	// close→send: op2.IsSelect=true → 跳过（send 不能做 Op2）
	// send→close: op1.IsSelect=true → 不跳过，但 MatchOpPair(send,close)=nil
	if len(co.SusConPairs) != 0 {
		t.Errorf("SusConPairs len = %d, want 0 (select prevents match)", len(co.SusConPairs))
	}
}

// ---------- TryEndPreExec ----------

func TestCorpusOp_TryEndPreExec(t *testing.T) {
	var phase uint32 = 1 // 预执行已结束
	co := NewCorpusOp(&phase)

	// 准备数据：先 Add op 信息，再构造 CorpusGort 带 covered pair
	opClose := makeOpInfo(1, 10, 0xABCD, feedback.OpTypeClose, feedback.OpKindChannel)
	opSend := makeOpInfo(2, 20, 0xABCD, feedback.OpTypeSend, feedback.OpKindChannel)
	co.ops[1] = opClose
	co.ops[2] = opSend
	co.byGid[10] = map[uint64]struct{}{1: {}}
	co.byGid[20] = map[uint64]struct{}{2: {}}

	cg := NewCorpusGort(&phase)
	cgPair := &feedback.GortPairInfo{
		Gid1: 10, Gid2: 20,
		CallLoc1:   feedback.CallLocationInfo{File: "a.go", Line: 1},
		CallLoc2:   feedback.CallLocationInfo{File: "b.go", Line: 2},
		IsObserved: true,
	}
	cg.CoveredConPairs[gortPairKey(cgPair)] = cgPair

	co.TryEndPreExec(cg)

	if !co.generated {
		t.Error("generated should be true after TryEndPreExec")
	}
	if len(co.SusConPairs) == 0 {
		t.Error("SusConPairs should be non-empty after generation")
	}
	if len(co.TryPairs) == 0 {
		t.Error("TryPairs should be filled after generation")
	}
}

// ---------- Get ----------

func TestCorpusOp_Get_BeforeGenerated(t *testing.T) {
	co := NewCorpusOp(new(uint32))

	if out := co.Get(); out != nil {
		t.Error("Get() should return nil before generation")
	}
}

func TestCorpusOp_Get_AfterGenerated(t *testing.T) {
	var phase uint32 = 1
	co := NewCorpusOp(&phase)
	co.generated = true

	// 直接手动填充 TryPairs
	opClose := makeOpInfo(1, 10, 0x100, feedback.OpTypeClose, feedback.OpKindChannel)
	opSend := makeOpInfo(2, 20, 0x100, feedback.OpTypeSend, feedback.OpKindChannel)
	pair := &feedback.OpPair{Op1: opClose, Op2: opSend, Danger: feedback.DangerCloseBeforeSend}
	co.TryPairs[opPairKey(pair)] = pair

	out := co.Get()
	if out == nil {
		t.Fatal("Get() should return non-nil after generation")
	}
	if len(out.TryPair) != 1 {
		t.Errorf("TryPair len = %d, want 1", len(out.TryPair))
	}
}

// ---------- OnGortCovered ----------

func TestCorpusOp_OnGortCovered(t *testing.T) {
	var phase uint32 = 1
	co := NewCorpusOp(&phase)
	co.generated = true

	// 准备 op 数据
	opClose := makeOpInfo(1, 10, 0xABCD, feedback.OpTypeClose, feedback.OpKindChannel)
	opSend := makeOpInfo(2, 20, 0xABCD, feedback.OpTypeSend, feedback.OpKindChannel)
	co.ops[1] = opClose
	co.ops[2] = opSend
	co.byGid[10] = map[uint64]struct{}{1: {}}
	co.byGid[20] = map[uint64]struct{}{2: {}}

	pairs := []*feedback.GortPairInfo{
		{Gid1: 10, Gid2: 20},
	}
	co.OnGortCovered(pairs)

	if len(co.SusConPairs) != 1 {
		t.Errorf("SusConPairs len = %d, want 1", len(co.SusConPairs))
	}
}

// ---------- opScore ----------

func TestCorpusOp_OpScore(t *testing.T) {
	var phase uint32 = 1
	co := NewCorpusOp(&phase)

	opClose := makeOpInfo(1, 10, 0x100, feedback.OpTypeClose, feedback.OpKindChannel)
	opSend := makeOpInfo(2, 20, 0x100, feedback.OpTypeSend, feedback.OpKindChannel)
	pair := &feedback.OpPair{Op1: opClose, Op2: opSend, Danger: feedback.DangerCloseBeforeSend}
	key := opPairKey(pair)
	co.SusConPairs[key] = pair

	// 基准分：0.5*10 - 0*2 = 5.0
	if score := co.opScore(key); score != 5.0 {
		t.Errorf("opScore() = %.2f, want 5.00", score)
	}

	// 有超时：5.0 - 2 = 3.0
	co.pairTimeouts[key] = 1
	if score := co.opScore(key); score != 3.0 {
		t.Errorf("opScore() after 1 timeout = %.2f, want 3.00", score)
	}

	// 不存在
	if score := co.opScore("nonexistent"); score != -1 {
		t.Errorf("opScore() for nonexistent = %.2f, want -1.00", score)
	}
}

// ---------- ApplySignals ----------

func TestCorpusOp_ApplySignals_Covered(t *testing.T) {
	var phase uint32 = 1
	co := NewCorpusOp(&phase)
	co.generated = true

	opClose := makeOpInfo(1, 10, 0x100, feedback.OpTypeClose, feedback.OpKindChannel)
	opSend := makeOpInfo(2, 20, 0x100, feedback.OpTypeSend, feedback.OpKindChannel)
	pair := &feedback.OpPair{Op1: opClose, Op2: opSend, Danger: feedback.DangerCloseBeforeSend}
	key := opPairKey(pair)
	co.SusConPairs[key] = pair
	co.TryPairs[key] = pair

	signals := []*feedback.CoverageSignal{
		{PreID: 1, NextID: 2, Success: true, Kind: feedback.SignalOpCovered},
	}
	co.ApplySignals(signals)

	// COVERED → SusConPairs → CoveredConPairs
	if _, ok := co.SusConPairs[key]; ok {
		t.Error("pair should be removed from SusConPairs after COVERED")
	}
	if _, ok := co.CoveredConPairs[key]; !ok {
		t.Error("pair should be in CoveredConPairs after COVERED")
	}
}

func TestCorpusOp_ApplySignals_TimeoutRemoval(t *testing.T) {
	var phase uint32 = 1
	co := NewCorpusOp(&phase)
	co.generated = true

	opClose := makeOpInfo(1, 10, 0x100, feedback.OpTypeClose, feedback.OpKindChannel)
	opSend := makeOpInfo(2, 20, 0x100, feedback.OpTypeSend, feedback.OpKindChannel)
	pair := &feedback.OpPair{Op1: opClose, Op2: opSend, Danger: feedback.DangerCloseBeforeSend}
	key := opPairKey(pair)
	co.SusConPairs[key] = pair
	co.TryPairs[key] = pair

	signal := &feedback.CoverageSignal{
		PreID: 1, NextID: 2, Success: false, Kind: feedback.SignalOpTimeout,
	}
	for i := 0; i < opMaxTimeouts; i++ {
		co.ApplySignals([]*feedback.CoverageSignal{signal})
		// 恢复 TryPairs
		if _, ok := co.SusConPairs[key]; ok {
			co.TryPairs[key] = pair
		}
	}

	if _, ok := co.SusConPairs[key]; ok {
		t.Error("pair should be removed from SusConPairs after max timeouts")
	}
	if _, ok := co.InfeasiblePairs[key]; !ok {
		t.Error("pair should be in InfeasiblePairs after max timeouts")
	}
}

func TestCorpusOp_ApplySignals_SelectNumExpansion(t *testing.T) {
	var phase uint32 = 1
	co := NewCorpusOp(&phase)
	co.generated = true

	opClose := makeOpInfo(1, 10, 0x100, feedback.OpTypeClose, feedback.OpKindChannel)
	opSend := makeOpInfo(2, 20, 0x100, feedback.OpTypeSend, feedback.OpKindChannel)
	pair := &feedback.OpPair{Op1: opClose, Op2: opSend, Danger: feedback.DangerCloseBeforeSend}
	key := opPairKey(pair)
	co.SusConPairs[key] = pair
	co.TryPairs[key] = pair
	origSelectNum := co.selectNum

	co.ApplySignals([]*feedback.CoverageSignal{
		{PreID: 1, NextID: 2, Success: false, Kind: feedback.SignalOpTimeout},
	})

	if co.selectNum != origSelectNum*2 {
		t.Errorf("selectNum = %d, want %d (doubled)", co.selectNum, origSelectNum*2)
	}
}

// ---------- RefillTryPairs ----------

func TestCorpusOp_RefillTryPairs(t *testing.T) {
	var phase uint32 = 1
	co := NewCorpusOp(&phase)
	co.generated = true

	// 手动填充 3 个 SusConPairs（不同超时）
	for i := 0; i < 3; i++ {
		a := makeOpInfo(uint64(i*2+1), 10, uint64(0x100+i), feedback.OpTypeClose, feedback.OpKindChannel)
		b := makeOpInfo(uint64(i*2+2), 20, uint64(0x100+i), feedback.OpTypeSend, feedback.OpKindChannel)
		pair := &feedback.OpPair{Op1: a, Op2: b, Danger: feedback.DangerCloseBeforeSend}
		key := opPairKey(pair)
		co.SusConPairs[key] = pair
		if i == 0 {
			co.pairTimeouts[key] = 3 // 降低优先级
		}
	}

	co.selectNum = 2
	co.RefillTryPairs()

	if len(co.TryPairs) != 2 {
		t.Errorf("TryPairs len = %d, want 2", len(co.TryPairs))
	}
}

// ---------- 边界 ----------

func TestCorpusOp_RefillTryPairs_Empty(t *testing.T) {
	var phase uint32 = 1
	co := NewCorpusOp(&phase)
	co.generated = true

	co.RefillTryPairs()
	if len(co.TryPairs) != 0 {
		t.Errorf("TryPairs len = %d, want 0 (empty SusConPairs)", len(co.TryPairs))
	}
}

func TestCorpusOp_TryEndPreExec_Idempotent(t *testing.T) {
	var phase uint32 = 1
	co := NewCorpusOp(&phase)
	co.generated = true // 已经生成过

	cg := NewCorpusGort(&phase)
	co.TryEndPreExec(cg) // 不应再次生成

	if len(co.SusConPairs) != 0 {
		t.Error("TryEndPreExec should be idempotent after generation")
	}
}
