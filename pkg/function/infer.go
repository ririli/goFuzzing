package function

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"time"
)

const funcEdgePrefix = "[FUNC_EDGE] "

// PairInfo 表示一对并发函数，对标 goroutine.GoroutinePairInfo
type PairInfo struct {
	FuncID1    uint64
	FuncID2    uint64
	CallLoc1   CallLocation
	CallLoc2   CallLocation
	Confidence float64
	SourceType string
	IsObserved bool
}

// pairKey 生成函数对的归一化去重键（较小 FuncID 在前）
func pairKey(fid1, fid2 uint64) string {
	if fid1 <= fid2 {
		return fmt.Sprintf("%d-%d", fid1, fid2)
	}
	return fmt.Sprintf("%d-%d", fid2, fid1)
}

// isTimeRangeOverlap 检测两个时间区间是否重叠
// 使用 <= 确保零时长（start==end）的瞬时调用也能被检测到并发
func isTimeRangeOverlap(start1, end1, start2, end2 int64) bool {
	if end1 == 0 || end2 == 0 {
		return false
	}
	return start1 <= end2 && start2 <= end1
}

// detectOverlaps 检测所有函数 ID 之间的时间重叠（Rule 0 直接观测）
// 对于每对函数 ID，检查其时间范围 [min(StartTime), max(EndTime)] 是否重叠
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
				continue // 跳过仍在运行的实例
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

// inferAdjacentPairs 基于COVERED对做 caller 方向的邻接推测（一跳）。
// Caller(A) × B, A × Caller(B)，置信度 0.5。
func (ft *FuncTracker) inferAdjacentPairs(observed []*PairInfo) []*PairInfo {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	// 构建 caller 查找表: callee -> caller
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

// sortFuncEdges 按 (Caller, Callee) 排序，保证输出确定性。
func sortFuncEdges(edges []funcEdge) {
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Caller != edges[j].Caller {
			return edges[i].Caller < edges[j].Caller
		}
		return edges[i].Callee < edges[j].Callee
	})
}

func writeFuncEdges(w io.Writer, edges []funcEdge) error {
	for _, edge := range edges {
		payload, err := json.Marshal(edge)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%s%s\n", funcEdgePrefix, payload); err != nil {
			return err
		}
	}
	return nil
}

// PrintFunctionPairs 打印所有函数并发对到stderr，对标 goroutine.PrintGoroutinePairs。
// 先输出当前执行的调用边，再输出直接观测（Rule 0）和 caller 邻接推测。
// funcID=0 不参与任何输出。跨推断函数间按 (fid1,fid2) 去重，冲突时保留高置信度。
// 格式：[COVERED] 或 [SUSPECT] fid1,fid2|file1:line1,file2:line2|confidence|sourceType;
func PrintFunctionPairs() {
	if skipRecord {
		return
	}
	time.Sleep(500 * time.Millisecond) // 等待异步调用执行完毕

	_ = writeFuncEdges(os.Stderr, tracker.snapshotEdges())

	// Rule 0: 直接观测的时间重叠对 → [COVERED]
	observed := tracker.detectOverlaps()

	// 观测锚定：caller 方向邻接推测（一跳） → [SUSPECT]
	adjacent := tracker.inferAdjacentPairs(observed)

	// 统一去重：按 (fid1,fid2) 合并，冲突时保留高置信度
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
