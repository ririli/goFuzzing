package goroutine

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"time"
)

const goroutineEdgePrefix = "[GORT_EDGE] "

// CallLocationInfo 表示goroutine创建位置信息
type CallLocationInfo struct {
	File string // 源文件路径
	Line int    // 行号
}

// GoroutinePairInfo 表示一对并发goroutine
type GoroutinePairInfo struct {
	Gid1       uint64           // 第一个goroutine的ID
	Gid2       uint64           // 第二个goroutine的ID
	CallLoc1   CallLocationInfo // 第一个go语句的位置
	CallLoc2   CallLocationInfo // 第二个go语句的位置
	Confidence float64          // 置信度（直接观测始终为1.0）
	SourceType string           // 来源类型（始终为"observed"）
}

// String 返回字符串表示（格式与callstack包一致，便于fuzzer解析）
// 置信度1.0输出[COVERED]，否则输出[SUSPECT]
func (g *GoroutinePairInfo) String() string {
	if g.Confidence >= 1.0 {
		return fmt.Sprintf("[COVERED] %d,%d|%s:%d,%s:%d|%.2f|%s;\n",
			g.Gid1, g.Gid2,
			g.CallLoc1.File, g.CallLoc1.Line,
			g.CallLoc2.File, g.CallLoc2.Line,
			g.Confidence, g.SourceType)
	}
	return fmt.Sprintf("[SUSPECT] %d,%d|%s:%d,%s:%d|%.2f|%s;\n",
		g.Gid1, g.Gid2,
		g.CallLoc1.File, g.CallLoc1.Line,
		g.CallLoc2.File, g.CallLoc2.Line,
		g.Confidence, g.SourceType)
}

// pairKeyAdj 生成标准化的去重key（小ID在前）
func pairKeyAdj(gid1, gid2 uint64) string {
	if gid1 <= gid2 {
		return fmt.Sprintf("%d-%d", gid1, gid2)
	}
	return fmt.Sprintf("%d-%d", gid2, gid1)
}

// isTimeRangeOverlap 检测两个时间区间是否重叠
// 使用 <= 确保零时长（start==end）的瞬时调用也能被检测到并发
func isTimeRangeOverlap(start1, end1, start2, end2 int64) bool {
	if end1 == 0 || end2 == 0 {
		return false
	}
	return start1 <= end2 && start2 <= end1
}

// DetectGoroutineOverlaps 检测所有goroutine ID之间的时间重叠
// 对于每对goroutine ID，检查是否有任何实例存在时间重叠
func (gt *GoroutineTracker) DetectGoroutineOverlaps() []*GoroutinePairInfo {
	gt.mu.RLock()
	defer gt.mu.RUnlock()

	// 1. 计算每个gid的时间范围: [min(StartTime), max(EndTime)]
	type gidRange struct {
		minStart int64
		maxEnd   int64
		callLoc  CallLocationInfo
	}

	ranges := make(map[uint64]*gidRange)
	for gid, instances := range gt.gortMap {
		var minStart, maxEnd int64
		first := true
		var loc CallLocationInfo
		for _, inst := range instances {
			if inst.EndTime == 0 {
				continue // 跳过仍在运行的实例
			}
			if first {
				minStart = inst.StartTime
				maxEnd = inst.EndTime
				loc = inst.CallLoc
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
			ranges[gid] = &gidRange{
				minStart: minStart,
				maxEnd:   maxEnd,
				callLoc:  loc,
			}
		}
	}

	// 2. O(n²) 两两比较
	gids := make([]uint64, 0, len(ranges))
	for gid := range ranges {
		gids = append(gids, gid)
	}
	sort.Slice(gids, func(i, j int) bool { return gids[i] < gids[j] })

	var overlaps []*GoroutinePairInfo
	for i := 0; i < len(gids); i++ {
		for j := i + 1; j < len(gids); j++ {
			gid1, gid2 := gids[i], gids[j]
			r1, r2 := ranges[gid1], ranges[gid2]

			if isTimeRangeOverlap(r1.minStart, r1.maxEnd, r2.minStart, r2.maxEnd) {
				overlaps = append(overlaps, &GoroutinePairInfo{
					Gid1:       gid1,
					Gid2:       gid2,
					CallLoc1:   r1.callLoc,
					CallLoc2:   r2.callLoc,
					Confidence: 1.0,
					SourceType: "observed",
				})
			}
		}
	}

	return overlaps
}

// InferAdjacentPairs 基于COVERED对做父子方向邻接推测。
// 向上：ga.Parent × gb, ga × gb.Parent  (conf=0.5)
//
//	goroutine父子是并发关系，parent可能提前退出，因此置信度适中
//
// 向下：ga.Children × gb, ga × gb.Children (conf=0.3)
//
//	子节点启动时机不确定，与重叠窗口的关系弱于parent方向
//
// gid=0参与扩展但不参与最终输出（由PrintGoroutinePairs过滤）
func (gt *GoroutineTracker) InferAdjacentPairs(observed []*GoroutinePairInfo) []*GoroutinePairInfo {
	gt.mu.RLock()
	defer gt.mu.RUnlock()

	seen := make(map[string]bool)
	var pairs []*GoroutinePairInfo

	for _, pair := range observed {
		// === 向下：查子节点 ===

		// A 的子与 B
		for _, child := range gt.childMap[pair.Gid1] {
			if child == pair.Gid2 {
				continue // 排除 (B, B) 自配对
			}
			key := pairKeyAdj(pair.Gid2, child)
			if seen[key] {
				continue
			}
			seen[key] = true
			pairs = append(pairs, &GoroutinePairInfo{
				Gid1:       child,
				Gid2:       pair.Gid2,
				Confidence: 0.3,
				SourceType: "inferred_adjacent",
			})
		}

		// B 的子与 A
		for _, child := range gt.childMap[pair.Gid2] {
			if child == pair.Gid1 {
				continue
			}
			key := pairKeyAdj(pair.Gid1, child)
			if seen[key] {
				continue
			}
			seen[key] = true
			pairs = append(pairs, &GoroutinePairInfo{
				Gid1:       pair.Gid1,
				Gid2:       child,
				Confidence: 0.3,
				SourceType: "inferred_adjacent",
			})
		}

		// === 向上：查父节点 ===

		// A 的父与 B
		if parentA := gt.getParentGid(pair.Gid1); parentA != 0 {
			if parentA != pair.Gid2 {
				key := pairKeyAdj(parentA, pair.Gid2)
				if !seen[key] {
					seen[key] = true
					pairs = append(pairs, &GoroutinePairInfo{
						Gid1:       parentA,
						Gid2:       pair.Gid2,
						Confidence: 0.5,
						SourceType: "inferred_adjacent",
					})
				}
			}
		}

		// B 的父与 A
		if parentB := gt.getParentGid(pair.Gid2); parentB != 0 {
			if parentB != pair.Gid1 {
				key := pairKeyAdj(pair.Gid1, parentB)
				if !seen[key] {
					seen[key] = true
					pairs = append(pairs, &GoroutinePairInfo{
						Gid1:       pair.Gid1,
						Gid2:       parentB,
						Confidence: 0.5,
						SourceType: "inferred_adjacent",
					})
				}
			}
		}
	}
	return pairs
}

// InferAllSiblingPairs 纯结构推断：同一parent的所有children之间两两配对。
// 依据：同一parent通过多个go语句spawn的goroutine在语义上就是并发的。
// 仅当parent拥有≥2个children时才产出对。
func (gt *GoroutineTracker) InferAllSiblingPairs() []*GoroutinePairInfo {
	gt.mu.RLock()
	defer gt.mu.RUnlock()

	seen := make(map[string]bool)
	var pairs []*GoroutinePairInfo

	for _, children := range gt.childMap {
		if len(children) < 2 {
			continue
		}
		for i := 0; i < len(children); i++ {
			for j := i + 1; j < len(children); j++ {
				key := pairKeyAdj(children[i], children[j])
				if seen[key] {
					continue
				}
				seen[key] = true
				pairs = append(pairs, &GoroutinePairInfo{
					Gid1:       children[i],
					Gid2:       children[j],
					Confidence: 0.5,
					SourceType: "inferred_sibling",
				})
			}
		}
	}
	return pairs
}

// InferSiblingAdjacentPairs 基于COVERED对做兄弟方向的邻接推测。
// ga-gb被观测到并发 → ga的兄弟 × gb, ga × gb的兄弟。
// sibling与ga的结构距离和parent相同（都是一条边），置信度0.5。
func (gt *GoroutineTracker) InferSiblingAdjacentPairs(observed []*GoroutinePairInfo) []*GoroutinePairInfo {
	gt.mu.RLock()
	defer gt.mu.RUnlock()

	seen := make(map[string]bool)
	var pairs []*GoroutinePairInfo

	for _, pair := range observed {
		// 方向一：ga的兄弟 × gb
		if parentA := gt.getParentGid(pair.Gid1); parentA != 0 {
			for _, sib := range gt.childMap[parentA] {
				if sib == pair.Gid1 || sib == pair.Gid2 {
					continue // 排除ga自身和gb，防止同父时产生自配对
				}
				key := pairKeyAdj(sib, pair.Gid2)
				if seen[key] {
					continue
				}
				seen[key] = true
				pairs = append(pairs, &GoroutinePairInfo{
					Gid1:       sib,
					Gid2:       pair.Gid2,
					Confidence: 0.5,
					SourceType: "inferred_sibling",
				})
			}
		}

		// 方向二：ga × gb的兄弟
		if parentB := gt.getParentGid(pair.Gid2); parentB != 0 {
			for _, sib := range gt.childMap[parentB] {
				if sib == pair.Gid1 || sib == pair.Gid2 {
					continue // 排除gb自身和ga，防止同父时产生自配对
				}
				key := pairKeyAdj(pair.Gid1, sib)
				if seen[key] {
					continue
				}
				seen[key] = true
				pairs = append(pairs, &GoroutinePairInfo{
					Gid1:       pair.Gid1,
					Gid2:       sib,
					Confidence: 0.5,
					SourceType: "inferred_sibling",
				})
			}
		}
	}
	return pairs
}

func writeGoroutineEdges(w io.Writer, edges []goroutineEdge) error {
	for _, edge := range edges {
		payload, err := json.Marshal(edge)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "%s%s\n", goroutineEdgePrefix, payload); err != nil {
			return err
		}
	}
	return nil
}

// PrintGoroutinePairs 打印所有goroutine并发对到stderr。
// 先输出当前执行的父子边，再输出直接观测（Rule 0）和三类推测
// （纯结构兄弟、邻接父子、邻接兄弟）。
// gid=0 不参与任何输出。跨推断函数间按 (gid1,gid2) 去重，冲突时保留高置信度。
// 格式：[COVERED] 或 [SUSPECT] gid1,gid2|file1:line1,file2:line2|confidence|sourceType;
func PrintGoroutinePairs() {
	if skipRecord {
		return
	}
	time.Sleep(500 * time.Millisecond) // 等待子goroutine执行完毕

	_ = writeGoroutineEdges(os.Stderr, tracker.snapshotEdges())

	// Rule 0: 直接观测的时间重叠对 → [COVERED]
	observedPairs := tracker.DetectGoroutineOverlaps()

	// 纯结构推断：同父的所有children两两配对 → [SUSPECT]
	allSiblingPairs := tracker.InferAllSiblingPairs()

	// 观测锚定：邻接推测（父子方向一跳） → [SUSPECT]
	adjacentPairs := tracker.InferAdjacentPairs(observedPairs)

	// 观测锚定：兄弟邻接推测（ga兄弟×gb, ga×gb兄弟） → [SUSPECT]
	siblingAdjacentPairs := tracker.InferSiblingAdjacentPairs(observedPairs)

	// 统一去重：按 (gid1,gid2) 合并，冲突时保留高置信度
	merged := make(map[string]*GoroutinePairInfo)
	for _, pair := range observedPairs {
		if pair.Gid1 == 0 || pair.Gid2 == 0 {
			continue
		}
		key := pairKeyAdj(pair.Gid1, pair.Gid2)
		if existing, ok := merged[key]; !ok || pair.Confidence > existing.Confidence {
			merged[key] = pair
		}
	}
	for _, pair := range allSiblingPairs {
		if pair.Gid1 == 0 || pair.Gid2 == 0 {
			continue
		}
		key := pairKeyAdj(pair.Gid1, pair.Gid2)
		if existing, ok := merged[key]; !ok || pair.Confidence > existing.Confidence {
			merged[key] = pair
		}
	}
	for _, pair := range adjacentPairs {
		if pair.Gid1 == 0 || pair.Gid2 == 0 {
			continue
		}
		key := pairKeyAdj(pair.Gid1, pair.Gid2)
		if existing, ok := merged[key]; !ok || pair.Confidence > existing.Confidence {
			merged[key] = pair
		}
	}
	for _, pair := range siblingAdjacentPairs {
		if pair.Gid1 == 0 || pair.Gid2 == 0 {
			continue
		}
		key := pairKeyAdj(pair.Gid1, pair.Gid2)
		if existing, ok := merged[key]; !ok || pair.Confidence > existing.Confidence {
			merged[key] = pair
		}
	}

	if len(merged) == 0 {
		print("No concurrent goroutine pairs found.\n")
		return
	}

	for _, pair := range merged {
		print(pair.String())
	}
}
