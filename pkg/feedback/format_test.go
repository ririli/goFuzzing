package feedback

import (
	"strings"
	"testing"
)

func TestParseFBOp_ChannelSend(t *testing.T) {
	line := "[FB]chan: obj=1234; opId=5; gid=10; op=send;"
	op, err := parseFBOp(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op.ObjKind != OpKindChannel {
		t.Errorf("ObjKind = %q, want %q", op.ObjKind, OpKindChannel)
	}
	if op.ObjAddr != 1234 {
		t.Errorf("ObjAddr = %d, want %d", op.ObjAddr, 1234)
	}
	if op.OpId != 5 {
		t.Errorf("OpId = %d, want %d", op.OpId, 5)
	}
	if op.Gid != 10 {
		t.Errorf("Gid = %d, want %d", op.Gid, 10)
	}
	if op.OpType != OpTypeSend {
		t.Errorf("OpType = %q, want %q", op.OpType, OpTypeSend)
	}
}

func TestParseFBOp_ChannelClose(t *testing.T) {
	line := "[FB]chan: obj=42; opId=1; gid=2; op=close;"
	op, err := parseFBOp(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op.ObjKind != OpKindChannel {
		t.Errorf("ObjKind = %q, want %q", op.ObjKind, OpKindChannel)
	}
	if op.OpType != OpTypeClose {
		t.Errorf("OpType = %q, want %q", op.OpType, OpTypeClose)
	}
}

func TestParseFBOp_WaitGroupAdd(t *testing.T) {
	line := "[FB]wg: obj=5678; opId=8; gid=12; op=add;"
	op, err := parseFBOp(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op.ObjKind != OpKindWaitGroup {
		t.Errorf("ObjKind = %q, want %q", op.ObjKind, OpKindWaitGroup)
	}
	if op.ObjAddr != 5678 {
		t.Errorf("ObjAddr = %d, want %d", op.ObjAddr, 5678)
	}
	if op.OpId != 8 {
		t.Errorf("OpId = %d, want %d", op.OpId, 8)
	}
	if op.Gid != 12 {
		t.Errorf("Gid = %d, want %d", op.Gid, 12)
	}
	if op.OpType != OpTypeAdd {
		t.Errorf("OpType = %q, want %q", op.OpType, OpTypeAdd)
	}
}

func TestParseFBOp_WaitGroupDone(t *testing.T) {
	line := "[FB]wg: obj=7777; opId=9; gid=15; op=done;"
	op, err := parseFBOp(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op.ObjKind != OpKindWaitGroup {
		t.Errorf("ObjKind = %q, want %q", op.ObjKind, OpKindWaitGroup)
	}
	if op.OpType != OpTypeDone {
		t.Errorf("OpType = %q, want %q", op.OpType, OpTypeDone)
	}
}

func TestParseFBOp_LargeIDs(t *testing.T) {
	line := "[FB]chan: obj=987842478084; opId=987842478097; gid=987842478084; op=send;"
	op, err := parseFBOp(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op.ObjAddr != 987842478084 {
		t.Errorf("ObjAddr = %d, want %d", op.ObjAddr, 987842478084)
	}
	if op.OpId != 987842478097 {
		t.Errorf("OpId = %d, want %d", op.OpId, 987842478097)
	}
	if op.Gid != 987842478084 {
		t.Errorf("Gid = %d, want %d", op.Gid, 987842478084)
	}
	if op.OpType != OpTypeSend {
		t.Errorf("OpType = %q, want %q", op.OpType, OpTypeSend)
	}
}

func TestParseFBOp_NoTrailingSemicolon(t *testing.T) {
	// 无尾部分号，容许解析
	line := "[FB]chan: obj=100; opId=1; gid=2; op=send"
	op, err := parseFBOp(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if op.OpType != OpTypeSend {
		t.Errorf("OpType = %q, want %q", op.OpType, OpTypeSend)
	}
	if op.OpId != 1 {
		t.Errorf("OpId = %d, want %d", op.OpId, 1)
	}
}

func TestParseFBOp_InvalidObjKind(t *testing.T) {
	line := "[FB]unknown: obj=100; opId=1; gid=2; op=send;"
	_, err := parseFBOp(line)
	if err == nil {
		t.Error("expected error for unknown object kind, got nil")
	}
}

func TestParseFBOp_NotFBPrefix(t *testing.T) {
	line := "[COVERED] 1,2|a.go:10,b.go:20|1.00|observed;"
	_, err := parseFBOp(line)
	if err == nil {
		t.Error("expected error for non-FB line, got nil")
	}
}

func TestParseFBOp_FuncStackAttribution(t *testing.T) {
	// function 粒度：fids 为操作发生时的函数栈（外层到内层）
	line := "[FB]chan: obj=1234; opId=5; gid=0; op=send; fids=3,7;"
	op, err := parseFBOp(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(op.FuncIDs) != 2 || op.FuncIDs[0] != 3 || op.FuncIDs[1] != 7 {
		t.Errorf("FuncIDs = %v, want [3 7]", op.FuncIDs)
	}
}

func TestParseFBOp_NoFidsBackwardCompatible(t *testing.T) {
	// goroutine 粒度的旧格式日志不带 fids，解析结果 FuncIDs 为空
	line := "[FB]wg: obj=5678; opId=8; gid=12; op=done;"
	op, err := parseFBOp(line)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(op.FuncIDs) != 0 {
		t.Errorf("FuncIDs = %v, want empty", op.FuncIDs)
	}
}

func TestParseFBOp_InvalidFids(t *testing.T) {
	line := "[FB]chan: obj=1; opId=2; gid=0; op=send; fids=3,abc;"
	_, err := parseFBOp(line)
	if err == nil {
		t.Error("expected error for invalid fids, got nil")
	}
}

func TestParseStdPairs_OnlyCovered(t *testing.T) {
	input := "[COVERED] 100,200|gopie/testdata/my.go:10,gopie/testdata/my.go:20|1.00|observed;"
	pairs, ops, err := ParseStdPairs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 1 {
		t.Fatalf("len(pairs) = %d, want 1", len(pairs))
	}
	if len(ops) != 0 {
		t.Fatalf("len(ops) = %d, want 0", len(ops))
	}
	p := pairs[0]
	if !p.IsObserved {
		t.Error("IsObserved should be true for [COVERED]")
	}
	if p.FuncID1 != 100 || p.FuncID2 != 200 {
		t.Errorf("FuncIDs = (%d,%d), want (100,200)", p.FuncID1, p.FuncID2)
	}
	if p.CallLoc1.File != "gopie/testdata/my.go" || p.CallLoc1.Line != 10 {
		t.Errorf("CallLoc1 = %s:%d", p.CallLoc1.File, p.CallLoc1.Line)
	}
	if p.CallLoc2.File != "gopie/testdata/my.go" || p.CallLoc2.Line != 20 {
		t.Errorf("CallLoc2 = %s:%d", p.CallLoc2.File, p.CallLoc2.Line)
	}
	if p.Confidence != 1.00 {
		t.Errorf("Confidence = %.2f, want 1.00", p.Confidence)
	}
	if p.SourceType != "observed" {
		t.Errorf("SourceType = %q, want %q", p.SourceType, "observed")
	}
}

func TestParseStdPairs_OnlySuspect(t *testing.T) {
	input := "[SUSPECT] 300,400|a/b/c.go:50, x/y/z.go:99|0.75|inferred_child1;"
	pairs, ops, err := ParseStdPairs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 1 {
		t.Fatalf("len(pairs) = %d, want 1", len(pairs))
	}
	if len(ops) != 0 {
		t.Fatalf("len(ops) = %d, want 0", len(ops))
	}
	p := pairs[0]
	if p.IsObserved {
		t.Error("IsObserved should be false for [SUSPECT]")
	}
	if p.FuncID1 != 300 || p.FuncID2 != 400 {
		t.Errorf("FuncIDs = (%d,%d), want (300,400)", p.FuncID1, p.FuncID2)
	}
	if p.Confidence != 0.75 {
		t.Errorf("Confidence = %.2f, want 0.75", p.Confidence)
	}
	if p.SourceType != "inferred_child1" {
		t.Errorf("SourceType = %q, want %q", p.SourceType, "inferred_child1")
	}
}

func TestParseStdPairs_OnlyFBOps(t *testing.T) {
	input := "[FB]chan: obj=111; opId=1; gid=10; op=send;\n[FB]wg: obj=222; opId=2; gid=20; op=add;"
	pairs, ops, err := ParseStdPairs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 0 {
		t.Fatalf("len(pairs) = %d, want 0", len(pairs))
	}
	if len(ops) != 2 {
		t.Fatalf("len(ops) = %d, want 2", len(ops))
	}

	if ops[0].ObjKind != OpKindChannel || ops[0].OpType != OpTypeSend {
		t.Errorf("ops[0] = %+v", ops[0])
	}
	if ops[1].ObjKind != OpKindWaitGroup || ops[1].OpType != OpTypeAdd {
		t.Errorf("ops[1] = %+v", ops[1])
	}
}

func TestParseStdPairs_MixedContent(t *testing.T) {
	input := strings.Join([]string{
		"[COVERED] 100,200|a.go:10,b.go:20|1.00|observed;",
		"[FB]chan: obj=111; opId=1; gid=10; op=send;",
		"[SUSPECT] 300,400|c.go:30,d.go:40|0.50|inferred_child1;",
		"[FB]wg: obj=222; opId=2; gid=20; op=done;",
		"[FB]chan: obj=333; opId=3; gid=30; op=close;",
	}, "\n")

	pairs, ops, err := ParseStdPairs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 2 {
		t.Fatalf("len(pairs) = %d, want 2", len(pairs))
	}
	if len(ops) != 3 {
		t.Fatalf("len(ops) = %d, want 3", len(ops))
	}

	// 验证 pairs
	if pairs[0].FuncID1 != 100 || !pairs[0].IsObserved {
		t.Errorf("pairs[0] = %+v", pairs[0])
	}
	if pairs[1].FuncID1 != 300 || pairs[1].IsObserved {
		t.Errorf("pairs[1] = %+v", pairs[1])
	}

	// 验证 ops
	expectedOps := []struct {
		kind OpKind
		typ  OpType
	}{
		{OpKindChannel, OpTypeSend},
		{OpKindWaitGroup, OpTypeDone},
		{OpKindChannel, OpTypeClose},
	}
	for i, exp := range expectedOps {
		if ops[i].ObjKind != exp.kind || ops[i].OpType != exp.typ {
			t.Errorf("ops[%d] = {%s, %s}, want {%s, %s}",
				i, ops[i].ObjKind, ops[i].OpType, exp.kind, exp.typ)
		}
	}
}

func TestParseStdPairs_EmptyString(t *testing.T) {
	pairs, ops, err := ParseStdPairs("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 0 {
		t.Errorf("len(pairs) = %d, want 0", len(pairs))
	}
	if len(ops) != 0 {
		t.Errorf("len(ops) = %d, want 0", len(ops))
	}
}

func TestParseStdPairs_BlankLines(t *testing.T) {
	input := "\n\n[FB]chan: obj=1; opId=1; gid=1; op=send;\n\n"
	pairs, ops, err := ParseStdPairs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 0 {
		t.Errorf("len(pairs) = %d, want 0", len(pairs))
	}
	if len(ops) != 1 {
		t.Errorf("len(ops) = %d, want 1", len(ops))
	}
}

func TestParseStdPairs_SkipsInvalidLines(t *testing.T) {
	input := "this is garbage\n[UNKNOWN] something\njust text\n[FB]chan: obj=1; opId=1; gid=1; op=send;"
	pairs, ops, err := ParseStdPairs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 0 {
		t.Errorf("len(pairs) = %d, want 0", len(pairs))
	}
	if len(ops) != 1 {
		t.Errorf("len(ops) = %d, want 1", len(ops))
	}
}

func TestParseStdPairs_WindowsPathWithColon(t *testing.T) {
	// 确保"文件路径中可能包含盘符 D:\..." 能正确解析行号
	input := "[COVERED] 1,2|D:\\gopie\\testdata\\my.go:10,E:\\other\\file.go:99|0.80|observed;"
	pairs, ops, err := ParseStdPairs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 1 {
		t.Fatalf("len(pairs) = %d, want 1", len(pairs))
	}
	p := pairs[0]
	if p.CallLoc1.File != "D:\\gopie\\testdata\\my.go" || p.CallLoc1.Line != 10 {
		t.Errorf("CallLoc1 = %s:%d", p.CallLoc1.File, p.CallLoc1.Line)
	}
	if p.CallLoc2.File != "E:\\other\\file.go" || p.CallLoc2.Line != 99 {
		t.Errorf("CallLoc2 = %s:%d", p.CallLoc2.File, p.CallLoc2.Line)
	}
	if len(ops) != 0 {
		t.Errorf("len(ops) = %d, want 0", len(ops))
	}
}

func TestParseGortEdges_MixedOutput(t *testing.T) {
	input := strings.Join([]string{
		"test log before protocol output",
		`[GORT_EDGE] {"parent":0,"child":10,"count":2}`,
		"[COVERED] 10,20|a.go:1,b.go:2|1.00|observed;",
		`[GORT_EDGE] {"parent":10,"child":20,"count":1}`,
	}, "\n")

	edges, err := ParseGortEdges(input)
	if err != nil {
		t.Fatalf("ParseGortEdges() error = %v", err)
	}
	if len(edges) != 2 {
		t.Fatalf("len(edges) = %d, want 2", len(edges))
	}
	if got := *edges[0]; got != (GortEdge{ParentGid: 0, ChildGid: 10, Count: 2}) {
		t.Errorf("edges[0] = %+v", got)
	}
	if got := *edges[1]; got != (GortEdge{ParentGid: 10, ChildGid: 20, Count: 1}) {
		t.Errorf("edges[1] = %+v", got)
	}
}

func TestParseGortEdges_ReturnsValidEdgesWithProtocolError(t *testing.T) {
	input := strings.Join([]string{
		`[GORT_EDGE] {"parent":0,"child":10,"count":2}`,
		"[GORT_EDGE] not-json",
		`[GORT_EDGE] {"parent":10,"child":20,"count":1}`,
	}, "\n")

	edges, err := ParseGortEdges(input)
	if err == nil {
		t.Fatal("ParseGortEdges() error = nil, want protocol error")
	}
	if len(edges) != 2 {
		t.Fatalf("len(edges) = %d, want 2", len(edges))
	}
	if edges[0].ChildGid != 10 || edges[1].ChildGid != 20 {
		t.Fatalf("edges = %+v", edges)
	}
}

func TestParseGortEdges_NoEdges(t *testing.T) {
	edges, err := ParseGortEdges("ordinary output\n[COVERED] 1,2|a.go:1,b.go:2|1.00|observed;")
	if err != nil {
		t.Fatalf("ParseGortEdges() error = %v", err)
	}
	if len(edges) != 0 {
		t.Fatalf("len(edges) = %d, want 0", len(edges))
	}
}

// ---------- ParseStdPairs 函数粒度补充 ----------

func TestParseStdPairs_CalltreeEmptyFileFormat(t *testing.T) {
	// calltree.PrintFunctionPairs 实际输出位置部分为 ":line"（无文件名）
	input := strings.Join([]string{
		"[COVERED] 1,2|:10,:20|1.00|observed;",
		"[SUSPECT] 1,3|:10,:30|0.50|inferred_adjacent;",
	}, "\n")

	pairs, ops, err := ParseStdPairs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 2 || len(ops) != 0 {
		t.Fatalf("ParseStdPairs() = %d pairs, %d ops; want 2, 0", len(pairs), len(ops))
	}
	p := pairs[0]
	if !p.IsObserved || p.Confidence != 1.0 {
		t.Errorf("pairs[0] = observed:%v confidence:%.2f, want true/1.00", p.IsObserved, p.Confidence)
	}
	if p.CallLoc1.File != "" || p.CallLoc1.Line != 10 {
		t.Errorf("CallLoc1 = %q:%d, want \"\":10", p.CallLoc1.File, p.CallLoc1.Line)
	}
	if p.CallLoc2.File != "" || p.CallLoc2.Line != 20 {
		t.Errorf("CallLoc2 = %q:%d, want \"\":20", p.CallLoc2.File, p.CallLoc2.Line)
	}
	if pairs[1].IsObserved || pairs[1].SourceType != "inferred_adjacent" {
		t.Errorf("pairs[1] = observed:%v source:%q", pairs[1].IsObserved, pairs[1].SourceType)
	}
}

// ---------- ParseGortPairs ----------

func TestParseGortPairs_Mixed(t *testing.T) {
	input := strings.Join([]string{
		"[COVERED] 10,20|main.go:5,main.go:9|1.00|observed;",
		"[SUSPECT] 10,30|main.go:5,main.go:12|0.50|inferred_sibling;",
		"[FB]chan: obj=1234; opId=5; gid=10; op=send;",
		"noise line",
	}, "\n")

	pairs, ops, err := ParseGortPairs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 2 {
		t.Fatalf("len(pairs) = %d, want 2", len(pairs))
	}
	if pairs[0].Gid1 != 10 || pairs[0].Gid2 != 20 || !pairs[0].IsObserved {
		t.Errorf("pairs[0] = %+v", pairs[0])
	}
	if pairs[1].SourceType != "inferred_sibling" || pairs[1].IsObserved {
		t.Errorf("pairs[1] = %+v", pairs[1])
	}
	if len(ops) != 1 || ops[0].OpId != 5 || ops[0].Gid != 10 {
		t.Fatalf("ops = %+v, want one op with opId=5 gid=10", ops)
	}
}

func TestParseGortPairs_Empty(t *testing.T) {
	pairs, ops, err := ParseGortPairs("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pairs) != 0 || len(ops) != 0 {
		t.Fatalf("ParseGortPairs(\"\") = %d pairs, %d ops; want 0, 0", len(pairs), len(ops))
	}
}

// ---------- ParseFuncEdges ----------

func TestParseFuncEdges_MixedOutput(t *testing.T) {
	input := strings.Join([]string{
		"test log before protocol output",
		`[FUNC_EDGE] {"caller":1,"callee":10,"count":3}`,
		"[COVERED] 1,2|:10,:20|1.00|observed;",
		`[FUNC_EDGE] {"caller":10,"callee":11,"count":1}`,
		`[GORT_EDGE] {"parent":0,"child":10,"count":2}`, // 异模式前缀忽略
	}, "\n")

	edges, err := ParseFuncEdges(input)
	if err != nil {
		t.Fatalf("ParseFuncEdges() error = %v", err)
	}
	if len(edges) != 2 {
		t.Fatalf("len(edges) = %d, want 2", len(edges))
	}
	if got := *edges[0]; got != (FuncEdge{Caller: 1, Callee: 10, Count: 3}) {
		t.Errorf("edges[0] = %+v", got)
	}
	if got := *edges[1]; got != (FuncEdge{Caller: 10, Callee: 11, Count: 1}) {
		t.Errorf("edges[1] = %+v", got)
	}
}

func TestParseFuncEdges_ReturnsValidEdgesWithProtocolError(t *testing.T) {
	input := strings.Join([]string{
		`[FUNC_EDGE] {"caller":1,"callee":10,"count":3}`,
		"[FUNC_EDGE] not-json",
	}, "\n")

	edges, err := ParseFuncEdges(input)
	if err == nil {
		t.Fatal("ParseFuncEdges() error = nil, want protocol error")
	}
	if len(edges) != 1 || edges[0].Callee != 10 {
		t.Fatalf("edges = %+v, want one valid edge", edges)
	}
}

// ---------- ParseSignals ----------

func TestParseSignals_PairAndOpSplits(t *testing.T) {
	input := strings.Join([]string{
		"{COVERED} {10, 20}",
		"{TIMEOUT} {30, 40}",
		"{COVERED_OP} {5, 6}",
		"{TIMEOUT_OP} {7, 8}",
		"noise",
	}, "\n")

	pairSignals, opSignals := ParseSignals(input)
	if len(pairSignals) != 2 || len(opSignals) != 2 {
		t.Fatalf("ParseSignals() = %d pair, %d op signals; want 2, 2", len(pairSignals), len(opSignals))
	}
	// {COVERED}/{TIMEOUT} 为两种粒度共用的中性信号
	if pairSignals[0].Kind != SignalPairCovered || !pairSignals[0].Success {
		t.Errorf("pairSignals[0] = %+v", pairSignals[0])
	}
	if pairSignals[1].Kind != SignalPairTimeout || pairSignals[1].Success {
		t.Errorf("pairSignals[1] = %+v", pairSignals[1])
	}
	if pairSignals[0].PreID != 10 || pairSignals[0].NextID != 20 {
		t.Errorf("pairSignals[0] ids = (%d,%d), want (10,20)", pairSignals[0].PreID, pairSignals[0].NextID)
	}
	if opSignals[0].Kind != SignalOpCovered || opSignals[1].Kind != SignalOpTimeout {
		t.Errorf("op signal kinds = %q/%q", opSignals[0].Kind, opSignals[1].Kind)
	}
}
