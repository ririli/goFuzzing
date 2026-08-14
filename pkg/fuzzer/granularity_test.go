package fuzzer

import (
	"testing"

	"toolkit/pkg/feedback"
)

// ---------- GetAdapter / 模式分发 ----------

func TestGetAdapter_ModeDispatch(t *testing.T) {
	if a := GetAdapter(ModeFunction); a.Mode != ModeFunction {
		t.Errorf("GetAdapter(ModeFunction).Mode = %q, want %q", a.Mode, ModeFunction)
	}
	if a := GetAdapter(ModeGoroutine); a.Mode != ModeGoroutine {
		t.Errorf("GetAdapter(ModeGoroutine).Mode = %q, want %q", a.Mode, ModeGoroutine)
	}
	// 未知模式回退 goroutine
	if a := GetAdapter(GranularityMode("unknown")); a.Mode != ModeGoroutine {
		t.Errorf("GetAdapter(unknown).Mode = %q, want goroutine fallback", a.Mode)
	}
}

func TestAdapter_NewCorpus_MatchesMode(t *testing.T) {
	var phase uint32

	if _, ok := GetAdapter(ModeGoroutine).NewCorpus(&phase).(*CorpusGort); !ok {
		t.Error("goroutine adapter should create *CorpusGort")
	}
	if _, ok := GetAdapter(ModeFunction).NewCorpus(&phase).(*CorpusFunc); !ok {
		t.Error("function adapter should create *CorpusFunc")
	}
}

// ---------- ParsePairs ----------

const gortStderrSample = "" +
	"[COVERED] 10,20|main.go:5,main.go:9|1.00|observed;\n" +
	"[SUSPECT] 10,30|main.go:5,main.go:12|0.50|inferred_sibling;\n" +
	"[FB]chan: obj=1234; opId=5; gid=10; op=send;\n"

func TestAdapter_ParsePairs_Goroutine(t *testing.T) {
	pairs, ops, err := GetAdapter(ModeGoroutine).ParsePairs(gortStderrSample)
	if err != nil {
		t.Fatalf("ParsePairs() error = %v", err)
	}
	if len(pairs) != 2 || len(ops) != 1 {
		t.Fatalf("ParsePairs() = %d pairs, %d ops; want 2, 1", len(pairs), len(ops))
	}
	gp, ok := pairs[0].(*feedback.GortPairInfo)
	if !ok {
		t.Fatalf("goroutine mode should yield *GortPairInfo, got %T", pairs[0])
	}
	if gp.ID1() != 10 || gp.ID2() != 20 || !gp.GetObserved() {
		t.Errorf("parsed gort pair = (%d,%d observed=%v), want (10,20,true)",
			gp.ID1(), gp.ID2(), gp.GetObserved())
	}
	if ops[0].OpId != 5 || ops[0].Gid != 10 {
		t.Errorf("parsed op = opId:%d gid:%d, want 5/10", ops[0].OpId, ops[0].Gid)
	}
}

// function.PrintFunctionPairs 的实际输出格式：位置部分为 ":line"（无文件名）
const funcStderrSample = "" +
	"[COVERED] 1,2|:10,:20|1.00|observed;\n" +
	"[SUSPECT] 1,3|:10,:30|0.50|inferred_adjacent;\n" +
	"[FB]wg: obj=5678; opId=8; gid=0; op=done;\n"

func TestAdapter_ParsePairs_Function(t *testing.T) {
	pairs, ops, err := GetAdapter(ModeFunction).ParsePairs(funcStderrSample)
	if err != nil {
		t.Fatalf("ParsePairs() error = %v", err)
	}
	if len(pairs) != 2 || len(ops) != 1 {
		t.Fatalf("ParsePairs() = %d pairs, %d ops; want 2, 1", len(pairs), len(ops))
	}
	fp, ok := pairs[0].(*feedback.SuspiciousPairInfo)
	if !ok {
		t.Fatalf("function mode should yield *SuspiciousPairInfo, got %T", pairs[0])
	}
	if fp.ID1() != 1 || fp.ID2() != 2 {
		t.Errorf("parsed func pair = (%d,%d), want (1,2)", fp.ID1(), fp.ID2())
	}
	// function 包输出的位置不含文件名，解析后 File 为空、Line 有效
	if fp.CallLocation1().File != "" || fp.CallLocation1().Line != 10 {
		t.Errorf("CallLocation1 = %+v, want {File:\"\" Line:10}", fp.CallLocation1())
	}
}

// ---------- ParseEdges ----------

func TestAdapter_ParseEdges_Goroutine(t *testing.T) {
	stderr := "[GORT_EDGE] {\"parent\":0,\"child\":10,\"count\":2}\n" +
		"some noise line\n"
	edges, err := GetAdapter(ModeGoroutine).ParseEdges(stderr)
	if err != nil {
		t.Fatalf("ParseEdges() error = %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("ParseEdges() = %d edges, want 1", len(edges))
	}
	if _, ok := edges[0].(*feedback.GortEdge); !ok {
		t.Fatalf("goroutine mode should yield *GortEdge, got %T", edges[0])
	}
	if edges[0].Parent() != 0 || edges[0].Child() != 10 || edges[0].GetCount() != 2 {
		t.Errorf("parsed edge = (%d->%d x%d), want (0->10 x2)",
			edges[0].Parent(), edges[0].Child(), edges[0].GetCount())
	}
}

func TestAdapter_ParseEdges_Function(t *testing.T) {
	stderr := "[FUNC_EDGE] {\"caller\":1,\"callee\":10,\"count\":3}\n" +
		"[GORT_EDGE] {\"parent\":0,\"child\":10,\"count\":2}\n" // 异模式前缀应被忽略
	edges, err := GetAdapter(ModeFunction).ParseEdges(stderr)
	if err != nil {
		t.Fatalf("ParseEdges() error = %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("ParseEdges() = %d edges, want 1", len(edges))
	}
	fe, ok := edges[0].(*feedback.FuncEdge)
	if !ok {
		t.Fatalf("function mode should yield *FuncEdge, got %T", edges[0])
	}
	if fe.Parent() != 1 || fe.Child() != 10 || fe.GetCount() != 3 {
		t.Errorf("parsed edge = (%d->%d x%d), want (1->10 x3)",
			fe.Parent(), fe.Child(), fe.GetCount())
	}
}

// ---------- OnPreExecEnd 双模式接线 ----------

func TestOnPreExecEnd_GoroutineGeneratesOpPairs(t *testing.T) {
	var phase uint32 = 1
	cg := NewCorpusGort(&phase)
	co := NewCorpusOp(&phase)

	// gid=1 上有 close，gid=2 上有 send，操作同一对象 → close-before-send 危险对
	co.Add([]*feedback.OpInfo{
		{OpId: 1, Gid: 1, ObjAddr: 0x100, OpType: feedback.OpTypeClose, ObjKind: feedback.OpKindChannel},
		{OpId: 2, Gid: 2, ObjAddr: 0x100, OpType: feedback.OpTypeSend, ObjKind: feedback.OpKindChannel},
	})

	covered := &feedback.GortPairInfo{Gid1: 1, Gid2: 2, Confidence: 1, IsObserved: true}
	cg.CoveredConPairs[feedback.GortPairKey(covered)] = covered

	cg.OnPreExecEnd(co)

	if !co.generated {
		t.Fatal("goroutine mode should mark OP corpus as generated")
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

func TestOnPreExecEnd_PhaseZeroIsNoop(t *testing.T) {
	var phase uint32 = 0
	cg := NewCorpusGort(&phase)
	cf := NewCorpusFunc(&phase)
	co := NewCorpusOp(&phase)

	cg.OnPreExecEnd(co)
	cf.OnPreExecEnd(co)

	if co.generated {
		t.Error("OnPreExecEnd must not generate before phase transition")
	}
}

func TestOnPreExecEnd_FunctionGeneratesOpPairs(t *testing.T) {
	var phase uint32 = 1
	cf := NewCorpusFunc(&phase)
	co := NewCorpusOp(&phase)

	// fids 归属：wg done 在函数 1 内，wg add 在函数 2 内
	co.Add([]*feedback.OpInfo{
		{OpId: 1, ObjAddr: 0x300, OpType: feedback.OpTypeDone, ObjKind: feedback.OpKindWaitGroup, FuncIDs: []uint64{1}},
		{OpId: 2, ObjAddr: 0x300, OpType: feedback.OpTypeAdd, ObjKind: feedback.OpKindWaitGroup, FuncIDs: []uint64{2}},
	})

	covered := &feedback.SuspiciousPairInfo{FuncID1: 1, FuncID2: 2, Confidence: 1, IsObserved: true}
	cf.CoveredConPairs[covered.PairKey()] = covered

	cf.OnPreExecEnd(co)

	if !co.generated {
		t.Fatal("function mode should mark OP corpus as generated")
	}
	if len(co.SusConPairs) != 1 {
		t.Fatalf("SusConPairs = %d, want 1 done-before-add pair", len(co.SusConPairs))
	}
	for _, p := range co.SusConPairs {
		if p.Danger != feedback.DangerDoneBeforeAdd {
			t.Errorf("danger = %q, want %q", p.Danger, feedback.DangerDoneBeforeAdd)
		}
	}
}

// ---------- 颗粒度字符串解析 ----------

func TestParseGranularity(t *testing.T) {
	cases := map[string]GranularityMode{
		"function":  ModeFunction,
		"func":      ModeFunction,
		"FUNCTION":  ModeFunction,
		" Func ":    ModeFunction,
		"goroutine": ModeGoroutine,
		"":          ModeGoroutine,
		"bogus":     ModeGoroutine,
	}
	for in, want := range cases {
		if got := ParseGranularity(in); got != want {
			t.Errorf("ParseGranularity(%q) = %q, want %q", in, got, want)
		}
	}
}
