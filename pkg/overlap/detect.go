package overlap

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"toolkit/pkg/calltree"
)

// OverlapAnalysis 时间重叠分析器
type OverlapAnalysis struct {
	mu        sync.Mutex
	collector *calltree.CallStackCollector
}

// NewOverlapAnalysis 创建重叠分析器
func NewOverlapAnalysis(collector *calltree.CallStackCollector) *OverlapAnalysis {
	return &OverlapAnalysis{
		collector: collector,
	}
}

// isTimeRangeOverlap 检测两个时间区间是否重叠
func isTimeRangeOverlap(start1, end1, start2, end2 int64) (bool, int64, int64) {
	if end1 == 0 || end2 == 0 {
		return false, 0, 0
	}
	if start1 <= end2 && start2 <= end1 {
		overlapStart := start1
		if start2 > overlapStart {
			overlapStart = start2
		}
		overlapEnd := end1
		if end2 < overlapEnd {
			overlapEnd = end2
		}
		return true, overlapStart, overlapEnd
	}
	return false, 0, 0
}

// collectNodes 辅助函数：收集树的所有节点
func collectNodes(node *calltree.FunctionCallNode, nodes *[]*calltree.FunctionCallNode) {
	if node == nil {
		return
	}
	*nodes = append(*nodes, node)
	for _, child := range node.Children {
		collectNodes(child, nodes)
	}
}

// normalizeFunctionPair 标准化函数对名称（确保顺序一致）
func normalizeFunctionPair(func1, func2 string) string {
	if func1 < func2 {
		return fmt.Sprintf("%s | %s", func1, func2)
	}
	return fmt.Sprintf("%s | %s", func2, func1)
}

// DetectFunctionOverlaps 检测所有goroutine之间的函数时间重叠
func (oa *OverlapAnalysis) DetectFunctionOverlaps() []*ConPairFunc {
	oa.mu.Lock()
	defer oa.mu.Unlock()

	trees := oa.collector.GetAllCallTrees()

	var allNodes []*calltree.FunctionCallNode
	for _, tree := range trees {
		if tree == nil {
			continue
		}
		collectNodes(tree, &allNodes)
	}

	// 过滤掉未结束的节点
	var completedNodes []*calltree.FunctionCallNode
	for _, node := range allNodes {
		if node.EndUnixNano != 0 {
			completedNodes = append(completedNodes, node)
		}
	}

	var overlaps []*ConPairFunc
	n := len(completedNodes)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			node1 := completedNodes[i]
			node2 := completedNodes[j]

			if node1.GoroutineID == node2.GoroutineID {
				continue
			}

			hasOverlap, overlapStart, overlapEnd := isTimeRangeOverlap(
				node1.StartUnixNano, node1.EndUnixNano,
				node2.StartUnixNano, node2.EndUnixNano,
			)

			if hasOverlap {
				overlapDuration := overlapEnd - overlapStart
				overlap := ConPairFunc{
					Node1: node1,
					Node2: node2,
					Overlap: TimeOverlap{
						OverlapStart:    overlapStart,
						OverlapEnd:      overlapEnd,
						OverlapDuration: overlapDuration,
						Func1Start:      node1.StartUnixNano,
						Func1End:        node1.EndUnixNano,
						Func2Start:      node2.StartUnixNano,
						Func2End:        node2.EndUnixNano,
					},
				}
				overlaps = append(overlaps, &overlap)
			}
		}
	}

	sort.Slice(overlaps, func(i, j int) bool {
		return overlaps[i].Overlap.OverlapDuration > overlaps[j].Overlap.OverlapDuration
	})

	return overlaps
}

// DetectGoroutineOverlaps 检测goroutine级别的重叠（更粗粒度）
func (oa *OverlapAnalysis) DetectGoroutineOverlaps() map[int][]int {
	trees := oa.collector.GetAllCallTrees()

	type gorange struct {
		Start int64
		End   int64
	}
	goroutineRanges := make(map[int]gorange)

	for goroutineID, tree := range trees {
		if tree == nil {
			continue
		}
		var nodes []*calltree.FunctionCallNode
		collectNodes(tree, &nodes)
		if len(nodes) == 0 {
			continue
		}
		start := nodes[0].StartUnixNano
		end := nodes[0].EndUnixNano
		for _, node := range nodes[1:] {
			if node.StartUnixNano < start {
				start = node.StartUnixNano
			}
			if node.EndUnixNano > end {
				end = node.EndUnixNano
			}
		}
		goroutineRanges[goroutineID] = gorange{Start: start, End: end}
	}

	goroutineOverlaps := make(map[int][]int)
	goroutineIDs := make([]int, 0, len(goroutineRanges))
	for id := range goroutineRanges {
		goroutineIDs = append(goroutineIDs, id)
	}

	for i := 0; i < len(goroutineIDs); i++ {
		for j := i + 1; j < len(goroutineIDs); j++ {
			gid1 := goroutineIDs[i]
			gid2 := goroutineIDs[j]
			range1 := goroutineRanges[gid1]
			range2 := goroutineRanges[gid2]

			hasOverlap, _, _ := isTimeRangeOverlap(
				range1.Start, range1.End,
				range2.Start, range2.End,
			)
			if hasOverlap {
				goroutineOverlaps[gid1] = append(goroutineOverlaps[gid1], gid2)
				goroutineOverlaps[gid2] = append(goroutineOverlaps[gid2], gid1)
			}
		}
	}
	return goroutineOverlaps
}

// FindConcurrentFunctionPairs 查找并发执行的函数对（按函数名分组）
func (oa *OverlapAnalysis) FindConcurrentFunctionPairs() map[string][]*ConPairFunc {
	overlaps := oa.DetectFunctionOverlaps()
	functionPairs := make(map[string][]*ConPairFunc)
	for _, overlap := range overlaps {
		key := normalizeFunctionPair(overlap.GetFunc1Name(), overlap.GetFunc2Name())
		functionPairs[key] = append(functionPairs[key], overlap)
	}
	return functionPairs
}

// AnalyzeFunctionConcurrency 检测特定函数的并发情况
func (oa *OverlapAnalysis) AnalyzeFunctionConcurrency(funcName string) []*ConPairFunc {
	allOverlaps := oa.DetectFunctionOverlaps()
	var result []*ConPairFunc
	for _, overlap := range allOverlaps {
		if overlap.GetFunc1Name() == funcName || overlap.GetFunc2Name() == funcName {
			result = append(result, overlap)
		}
	}
	return result
}

// GetTimelineData 获取时间线可视化数据
func (oa *OverlapAnalysis) GetTimelineData() []struct {
	GoroutineID int
	FuncName    string
	StartTime   int64
	EndTime     int64
} {
	trees := oa.collector.GetAllCallTrees()

	var timeline []struct {
		GoroutineID int
		FuncName    string
		StartTime   int64
		EndTime     int64
	}

	for goroutineID, tree := range trees {
		if tree == nil {
			continue
		}
		var nodes []*calltree.FunctionCallNode
		collectNodes(tree, &nodes)
		for _, node := range nodes {
			if node.EndUnixNano != 0 {
				timeline = append(timeline, struct {
					GoroutineID int
					FuncName    string
					StartTime   int64
					EndTime     int64
				}{
					GoroutineID: goroutineID,
					FuncName:    node.FuncName,
					StartTime:   node.StartUnixNano,
					EndTime:     node.EndUnixNano,
				})
			}
		}
	}

	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].StartTime < timeline[j].StartTime
	})

	return timeline
}

// PrintOverlapReport 打印重叠分析报告
func (oa *OverlapAnalysis) PrintOverlapReport() {
	fmt.Println("=== 函数执行时间重叠分析报告 ===")
	fmt.Println()

	overlaps := oa.DetectFunctionOverlaps()
	fmt.Printf("检测到 %d 个跨goroutine的函数时间重叠\n", len(overlaps))
	fmt.Println()

	if len(overlaps) > 0 {
		fmt.Println("详细重叠信息（按重叠时长排序）:")
		fmt.Println(strings.Repeat("-", 120))
		for i, overlap := range overlaps {
			fmt.Printf("%d. %s (Goroutine-%d, CallID:%d) 与 %s (Goroutine-%d, CallID:%d)\n",
				i+1, overlap.GetFunc1Name(), overlap.GetGoroutine1(), overlap.GetCallID1(),
				overlap.GetFunc2Name(), overlap.GetGoroutine2(), overlap.GetCallID2())
			fmt.Printf("   重叠时间: %v ~ %v (时长: %v)\n",
				overlap.Overlap.OverlapStart,
				overlap.Overlap.OverlapEnd,
				overlap.Overlap.OverlapDuration)
			fmt.Printf("   函数1执行: %v ~ %v\n",
				overlap.Overlap.Func1Start,
				overlap.Overlap.Func1End)
			fmt.Printf("   函数2执行: %v ~ %v\n",
				overlap.Overlap.Func2Start,
				overlap.Overlap.Func2End)
			fmt.Println()
		}
	}

	// goroutine级别重叠
	goroutineOverlaps := oa.DetectGoroutineOverlaps()
	fmt.Println("=== Goroutine级别重叠分析 ===")
	fmt.Printf("共有 %d 个goroutine存在时间重叠\n", len(goroutineOverlaps))
	fmt.Println()
	for gid, overlappingGids := range goroutineOverlaps {
		fmt.Printf("Goroutine-%d 与以下goroutine重叠: ", gid)
		for i, ogid := range overlappingGids {
			if i > 0 {
				fmt.Print(", ")
			}
			fmt.Printf("Goroutine-%d", ogid)
		}
		fmt.Println()
	}
	fmt.Println()

	// 常见的并发函数对
	functionPairs := oa.FindConcurrentFunctionPairs()
	fmt.Println("=== 常见的并发函数对 ===")
	for pair, pairOverlaps := range functionPairs {
		fmt.Printf("函数对: %s\n", pair)
		fmt.Printf("  重叠次数: %d\n", len(pairOverlaps))
		var totalDuration int64
		for _, o := range pairOverlaps {
			totalDuration += o.Overlap.OverlapDuration
		}
		avgDuration := totalDuration / int64(len(pairOverlaps))
		fmt.Printf("  平均重叠时长: %v\n", avgDuration)
		fmt.Println()
	}
}

// PrintConcurrencyAnalysis 包级别便捷函数
func PrintConcurrencyAnalysis(collector *calltree.CallStackCollector) {
	analyzer := NewOverlapAnalysis(collector)
	analyzer.PrintOverlapReport()
}
