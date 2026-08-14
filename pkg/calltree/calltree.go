// Package calltree provides function call tree tracking for function-level fuzzing.
// It records function entry/exit times, detects time-range overlaps,
// and outputs [COVERED]/[SUSPECT] pairs and [FUNC_EDGE] topology edges to stderr.
package calltree

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"
)

// CallLocation records where a function call originated.
type CallLocation struct {
	File string
	Line int
}

// FuncRecord records a single function call instance lifecycle.
type FuncRecord struct {
	FuncID    uint64
	CallID    uint64
	ParentID  uint64
	StartTime int64 // UnixNano
	EndTime   int64 // UnixNano, 0 = still running
	CallLoc   CallLocation
}

// funcEdge represents a caller-callee edge aggregated from one execution.
type funcEdge struct {
	Caller uint64 `json:"caller"`
	Callee uint64 `json:"callee"`
	Count  uint64 `json:"count"`
}

// FuncTracker collects function call lifecycle records.
type FuncTracker struct {
	mu         sync.RWMutex
	funcMap    map[uint64][]*FuncRecord // funcID -> instances
	callerMap  map[uint64]uint64        // callee funcID -> caller funcID (most recent)
	nextCallID uint64
	depth      map[int][]*FuncRecord // goroutine ID -> call stack (for parent tracking)
}

var (
	tracker    *FuncTracker
	skipRecord bool
)

func init() {
	tracker = NewFuncTracker()
	skipRecord = os.Getenv("RECORD_STACK") == "1"
}

// NewFuncTracker creates a new function call tracker.
func NewFuncTracker() *FuncTracker {
	return &FuncTracker{
		funcMap:   make(map[uint64][]*FuncRecord),
		callerMap: make(map[uint64]uint64),
		depth:     make(map[int][]*FuncRecord),
	}
}

// getGoroutineID returns the current OS goroutine ID.
func getGoroutineID() int {
	buf := make([]byte, 64)
	n := runtime.Stack(buf, false)
	var id int
	if n > 0 {
		fmt.Sscanf(string(buf[:n]), "goroutine %d", &id)
	}
	return id
}

// Trace is called as defer Trace(funcID)() at function entry.
// It records the function's lifecycle and participates in barrier control
// via breakpoint.PointControl.
func Trace(funcID uint64) func() {
	// We can't import breakpoint here to avoid circular deps;
	// PointControl is called from the instrumentation directly.
	if skipRecord {
		return func() {}
	}
	gid := getGoroutineID()
	startTime := time.Now().UnixNano()

	tracker.mu.Lock()
	callID := tracker.nextCallID
	tracker.nextCallID++

	// Determine parent (caller) from the call stack
	var parentID uint64
	stack := tracker.depth[gid]
	if len(stack) > 0 {
		parentID = stack[len(stack)-1].FuncID
	}
	tracker.callerMap[funcID] = parentID

	rec := &FuncRecord{
		FuncID:    funcID,
		CallID:    callID,
		ParentID:  parentID,
		StartTime: startTime,
	}
	tracker.funcMap[funcID] = append(tracker.funcMap[funcID], rec)
	tracker.depth[gid] = append(tracker.depth[gid], rec)
	tracker.mu.Unlock()

	return func() {
		if skipRecord {
			return
		}
		tracker.mu.Lock()
		rec.EndTime = time.Now().UnixNano()
		// Pop from stack
		s := tracker.depth[gid]
		for i := len(s) - 1; i >= 0; i-- {
			if s[i].CallID == callID {
				tracker.depth[gid] = s[:i]
				break
			}
		}
		tracker.mu.Unlock()
	}
}

// CurrentFuncStack 返回当前 OS goroutine 调用栈上所有未退出的被插桩函数 ID。
// 供 sched 包在 [FB] 日志中归属操作所属函数：一个操作归属于栈上全部函数，
// 对应 goroutine 粒度下"操作归属于所在协程"的语义。
// funcID=0（主测试函数 EnterMain 压栈）不参与归属。
func CurrentFuncStack() []uint64 {
	gid := getGoroutineID()
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	stack := tracker.depth[gid]
	fids := make([]uint64, 0, len(stack))
	for _, rec := range stack {
		if rec.FuncID != 0 {
			fids = append(fids, rec.FuncID)
		}
	}
	return fids
}

// EnterMain records the main test function entry.
func EnterMain() {
	if skipRecord {
		return
	}
	gid := getGoroutineID()
	tracker.mu.Lock()
	rec := &FuncRecord{
		FuncID:    0,
		CallID:    tracker.nextCallID,
		ParentID:  0,
		StartTime: time.Now().UnixNano(),
	}
	tracker.nextCallID++
	tracker.funcMap[0] = append(tracker.funcMap[0], rec)
	tracker.depth[gid] = append(tracker.depth[gid], rec)
	tracker.mu.Unlock()
}

// ExitMain records the main test function exit.
func ExitMain() {
	if skipRecord {
		return
	}
	gid := getGoroutineID()
	tracker.mu.Lock()
	if instances := tracker.funcMap[0]; len(instances) > 0 {
		instances[len(instances)-1].EndTime = time.Now().UnixNano()
	}
	tracker.depth[gid] = nil
	tracker.mu.Unlock()
}

// snapshotEdges returns counted caller-callee edges from this execution.
func (ft *FuncTracker) snapshotEdges() []funcEdge {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	type edgeKey struct {
		caller uint64
		callee uint64
	}
	counts := make(map[edgeKey]uint64)
	for callee, caller := range ft.callerMap {
		if callee == 0 || caller == callee {
			continue
		}
		counts[edgeKey{caller: caller, callee: callee}]++
	}

	edges := make([]funcEdge, 0, len(counts))
	for key, count := range counts {
		edges = append(edges, funcEdge{
			Caller: key.caller,
			Callee: key.callee,
			Count:  count,
		})
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Caller != edges[j].Caller {
			return edges[i].Caller < edges[j].Caller
		}
		return edges[i].Callee < edges[j].Callee
	})
	return edges
}

const funcEdgePrefix = "[FUNC_EDGE] "

func writeFuncEdges(edges []funcEdge) {
	for _, edge := range edges {
		payload, _ := json.Marshal(edge)
		fmt.Fprintf(os.Stderr, "%s%s\n", funcEdgePrefix, payload)
	}
}

// PairInfo represents a pair of concurrent function IDs.
type PairInfo struct {
	FuncID1    uint64
	FuncID2    uint64
	CallLoc1   CallLocation
	CallLoc2   CallLocation
	Confidence float64
	SourceType string
	IsObserved bool
}

// pairKey generates a normalized dedup key for a function pair.
func pairKey(gid1, gid2 uint64) string {
	if gid1 <= gid2 {
		return fmt.Sprintf("%d-%d", gid1, gid2)
	}
	return fmt.Sprintf("%d-%d", gid2, gid1)
}

// isTimeRangeOverlap detects if two time ranges overlap.
func isTimeRangeOverlap(start1, end1, start2, end2 int64) bool {
	if end1 == 0 || end2 == 0 {
		return false
	}
	return start1 <= end2 && start2 <= end1
}

// detectOverlaps finds all function ID pairs with time-range overlap.
func (ft *FuncTracker) detectOverlaps() []*PairInfo {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	type idRange struct {
		minStart int64
		maxEnd   int64
	}
	ranges := make(map[uint64]*idRange)
	for fid, instances := range ft.funcMap {
		var minStart, maxEnd int64
		first := true
		for _, inst := range instances {
			if inst.EndTime == 0 {
				continue
			}
			if first {
				minStart = inst.StartTime
				maxEnd = inst.EndTime
				first = false
			} else {
				if inst.StartTime < minStart {
					minStart = inst.StartTime
				}
				if inst.EndTime > maxEnd {
					maxEnd = inst.EndTime
				}
			}
		}
		if !first {
			ranges[fid] = &idRange{minStart: minStart, maxEnd: maxEnd}
		}
	}

	fids := make([]uint64, 0, len(ranges))
	for fid := range ranges {
		fids = append(fids, fid)
	}
	sort.Slice(fids, func(i, j int) bool { return fids[i] < fids[j] })

	var overlaps []*PairInfo
	for i := 0; i < len(fids); i++ {
		for j := i + 1; j < len(fids); j++ {
			fid1, fid2 := fids[i], fids[j]
			r1, r2 := ranges[fid1], ranges[fid2]
			if isTimeRangeOverlap(r1.minStart, r1.maxEnd, r2.minStart, r2.maxEnd) {
				overlaps = append(overlaps, &PairInfo{
					FuncID1:    fid1,
					FuncID2:    fid2,
					Confidence: 1.0,
					SourceType: "observed",
					IsObserved: true,
				})
			}
		}
	}
	return overlaps
}

// inferAdjacentPairs infers pairs adjacent to observed pairs via caller-callee edges.
func (ft *FuncTracker) inferAdjacentPairs(observed []*PairInfo) []*PairInfo {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	// Build caller lookup: callee -> caller
	callerOf := make(map[uint64]uint64)
	for callee, caller := range ft.callerMap {
		callerOf[callee] = caller
	}

	seen := make(map[string]bool)
	var pairs []*PairInfo

	for _, p := range observed {
		// Caller(A) x B
		if caller, ok := callerOf[p.FuncID1]; ok && caller != 0 && caller != p.FuncID2 {
			key := pairKey(caller, p.FuncID2)
			if !seen[key] {
				seen[key] = true
				pairs = append(pairs, &PairInfo{
					FuncID1: caller, FuncID2: p.FuncID2,
					Confidence: 0.5, SourceType: "inferred_adjacent", IsObserved: false,
				})
			}
		}
		// A x Caller(B)
		if caller, ok := callerOf[p.FuncID2]; ok && caller != 0 && caller != p.FuncID1 {
			key := pairKey(p.FuncID1, caller)
			if !seen[key] {
				seen[key] = true
				pairs = append(pairs, &PairInfo{
					FuncID1: p.FuncID1, FuncID2: caller,
					Confidence: 0.5, SourceType: "inferred_adjacent", IsObserved: false,
				})
			}
		}
	}
	return pairs
}

// PrintFunctionPairs outputs function concurrent pairs and topology edges to stderr.
// Format matches goroutine mode: [COVERED]/[SUSPECT] and [FUNC_EDGE].
func PrintFunctionPairs() {
	if skipRecord {
		return
	}
	time.Sleep(500 * time.Millisecond)

	writeFuncEdges(tracker.snapshotEdges())

	observed := tracker.detectOverlaps()
	adjacent := tracker.inferAdjacentPairs(observed)

	// Merge
	merged := make(map[string]*PairInfo)
	for _, p := range observed {
		if p.FuncID1 == 0 || p.FuncID2 == 0 {
			continue
		}
		key := pairKey(p.FuncID1, p.FuncID2)
		if existing, ok := merged[key]; !ok || p.Confidence > existing.Confidence {
			merged[key] = p
		}
	}
	for _, p := range adjacent {
		if p.FuncID1 == 0 || p.FuncID2 == 0 {
			continue
		}
		key := pairKey(p.FuncID1, p.FuncID2)
		if existing, ok := merged[key]; !ok || p.Confidence > existing.Confidence {
			merged[key] = p
		}
	}

	for _, p := range merged {
		if p.IsObserved {
			fmt.Fprintf(os.Stderr, "[COVERED] %d,%d|:%d,:%d|%.2f|%s;\n",
				p.FuncID1, p.FuncID2,
				p.CallLoc1.Line, p.CallLoc2.Line,
				p.Confidence, p.SourceType)
		} else {
			fmt.Fprintf(os.Stderr, "[SUSPECT] %d,%d|:%d,:%d|%.2f|%s;\n",
				p.FuncID1, p.FuncID2,
				p.CallLoc1.Line, p.CallLoc2.Line,
				p.Confidence, p.SourceType)
		}
	}
}
