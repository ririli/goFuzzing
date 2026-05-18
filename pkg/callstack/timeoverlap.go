package callstack

import (
	"fmt"
	"sort"
	"strings"
)

// TimeOverlap 表示ConPair两个函数执行时重叠的时间
type TimeOverlap struct {
	OverlapStart    int64 // 重叠开始时间
	OverlapEnd      int64 // 重叠结束时间
	OverlapDuration int64 // 重叠时长
	Func1Start      int64 // 函数1开始时间
	Func1End        int64 // 函数1结束时间
	Func2Start      int64 // 函数2开始时间
	Func2End        int64 // 函数2结束时间
}

// ConPairFunc 表示两个函数执行时间重叠的信息
type ConPairFunc struct {
	Node1   *FunctionCallNode // 第一个函数节点（直接指针访问）
	Node2   *FunctionCallNode // 第二个函数节点（直接指针访问）
	Overlap TimeOverlap       // 时间重叠信息
}

// GetFunc1Name 获取第一个函数名
func (c ConPairFunc) GetFunc1Name() string {
	if c.Node1 != nil {
		return c.Node1.FuncName
	}
	return "unknown"
}

// GetFunc2Name 获取第二个函数名
func (c ConPairFunc) GetFunc2Name() string {
	if c.Node2 != nil {
		return c.Node2.FuncName
	}
	return "unknown"
}

// GetCallID1 获取第一个调用ID
func (c ConPairFunc) GetCallID1() uint64 {
	if c.Node1 != nil {
		return c.Node1.CallID
	}
	return 0
}

// GetCallID2 获取第二个调用ID
func (c ConPairFunc) GetCallID2() uint64 {
	if c.Node2 != nil {
		return c.Node2.CallID
	}
	return 0
}

// GetGoroutine1 获取第一个goroutine ID
func (c ConPairFunc) GetGoroutine1() int {
	if c.Node1 != nil {
		return c.Node1.GoroutineID
	}
	return 0
}

// GetGoroutine2 获取第二个goroutine ID
func (c ConPairFunc) GetGoroutine2() int {
	if c.Node2 != nil {
		return c.Node2.GoroutineID
	}
	return 0
}

// GetFuncID1 获取第一个函数ID
func (c ConPairFunc) GetFuncID1() uint64 {
	if c.Node1 != nil {
		return c.Node1.FuncID
	}
	return 0
}

// GetFuncID2 获取第二个函数ID
func (c ConPairFunc) GetFuncID2() uint64 {
	if c.Node2 != nil {
		return c.Node2.FuncID
	}
	return 0
}

// String 返回字符串表示
func (c ConPairFunc) String() string {
	return fmt.Sprintf("%s@%d (CallID=%d) ↔ %s@%d (CallID=%d), Overlap=%dns",
		c.GetFunc1Name(), c.GetGoroutine1(), c.GetCallID1(),
		c.GetFunc2Name(), c.GetGoroutine2(), c.GetCallID2(),
		c.Overlap.OverlapDuration)
}

// OverlapAnalysis 时间重叠分析器
type OverlapAnalysis struct {
	collector *CallStackCollector
}

// NewOverlapAnalysis 创建重叠分析器
func NewOverlapAnalysis(collector *CallStackCollector) *OverlapAnalysis {
	return &OverlapAnalysis{
		collector: collector,
	}
}

// 检测两个时间区间是否重叠
func isTimeRangeOverlap(start1, end1, start2, end2 int64) (bool, int64, int64) {
	// 检查是否重叠：start1 < end2 && start2 < end1
	if end1 == 0 || end2 == 0 {
		// 如果任一区间还没有结束，暂时视为不重叠或特殊处理
		return false, 0, 0
	}

	if start1 < end2 && start2 < end1 {
		// 计算重叠区间
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

// DetectFunctionOverlaps 检测所有goroutine之间的函数时间重叠
func (oa *OverlapAnalysis) DetectFunctionOverlaps() []*ConPairFunc {
	// 获取所有goroutine的调用树
	trees := oa.collector.GetAllCallTrees()

	// 收集所有已结束的函数调用节点
	var allNodes []*FunctionCallNode
	for _, tree := range trees {
		if tree == nil {
			continue
		}
		// todo 递归调用如果过深，可能栈溢出
		// 遍历树收集所有节点
		oa.collectNodes(tree, &allNodes)
	}

	// 过滤掉未结束的节点（EndTime为零值）
	var completedNodes []*FunctionCallNode
	for _, node := range allNodes {
		if node.EndUnixNano != 0 {
			completedNodes = append(completedNodes, node)
		}
	}

	// 检测重叠
	var overlaps []*ConPairFunc
	n := len(completedNodes)
	// todo 时间复杂度很高，待后续优化
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			node1 := completedNodes[i]
			node2 := completedNodes[j]

			// 只检查不同goroutine的函数
			if node1.GoroutineID == node2.GoroutineID {
				continue
			}

			// 检查时间重叠
			hasOverlap, overlapStart, overlapEnd := isTimeRangeOverlap(
				node1.StartUnixNano, node1.EndUnixNano,
				node2.StartUnixNano, node2.EndUnixNano,
			)

			if hasOverlap {
				overlapDuration := overlapEnd - overlapStart

				// 直接存储节点指针，不再需要逐个字段复制
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

	// 按重叠时长降序排序
	sort.Slice(overlaps, func(i, j int) bool {
		return overlaps[i].Overlap.OverlapDuration > overlaps[j].Overlap.OverlapDuration
	})

	return overlaps
}

// 辅助函数：收集树的所有节点
func (oa *OverlapAnalysis) collectNodes(node *FunctionCallNode, nodes *[]*FunctionCallNode) {
	if node == nil {
		return
	}

	*nodes = append(*nodes, node)
	for _, child := range node.Children {
		oa.collectNodes(child, nodes)
	}
}

// DetectGoroutineOverlaps 检测goroutine级别的重叠（更粗粒度）
func (oa *OverlapAnalysis) DetectGoroutineOverlaps() map[int][]int {
	trees := oa.collector.GetAllCallTrees()

	// 计算每个goroutine的时间范围（基于其所有函数调用）
	goroutineRanges := make(map[int]struct {
		Start int64
		End   int64
		Nodes []*FunctionCallNode
	})

	for goroutineID, tree := range trees {
		if tree == nil {
			continue
		}

		var nodes []*FunctionCallNode
		oa.collectNodes(tree, &nodes)

		if len(nodes) == 0 {
			continue
		}

		// 找到goroutine中最早开始和最晚结束的时间
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

		goroutineRanges[goroutineID] = struct {
			Start int64
			End   int64
			Nodes []*FunctionCallNode
		}{
			Start: start,
			End:   end,
			Nodes: nodes,
		}
	}

	// 检测goroutine之间的重叠
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

	// 按函数对分组
	functionPairs := make(map[string][]*ConPairFunc)

	for _, overlap := range overlaps {
		// 创建标准化的键（确保相同的函数对总是以相同的方式排序）
		key := oa.normalizeFunctionPair(overlap.GetFunc1Name(), overlap.GetFunc2Name())

		functionPairs[key] = append(functionPairs[key], overlap)
	}

	return functionPairs
}

// 标准化函数对名称（确保顺序一致）
func (oa *OverlapAnalysis) normalizeFunctionPair(func1, func2 string) string {
	if func1 < func2 {
		return fmt.Sprintf("%s | %s", func1, func2)
	}
	return fmt.Sprintf("%s | %s", func2, func1)
}

// PrintOverlapReport 打印重叠分析报告
func (oa *OverlapAnalysis) PrintOverlapReport() {
	fmt.Println("=== 函数执行时间重叠分析报告 ===")
	fmt.Println()

	// 1. 检测函数级别重叠
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

	// 2. 检测goroutine级别重叠
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

	// 3. 分析常见的并发函数对
	functionPairs := oa.FindConcurrentFunctionPairs()

	fmt.Println("=== 常见的并发函数对 ===")
	for pair, pairOverlaps := range functionPairs {
		fmt.Printf("函数对: %s\n", pair)
		fmt.Printf("  重叠次数: %d\n", len(pairOverlaps))

		// 计算平均重叠时长
		var totalDuration int64
		for _, o := range pairOverlaps {
			totalDuration += o.Overlap.OverlapDuration
		}
		avgDuration := totalDuration / int64(len(pairOverlaps))

		fmt.Printf("  平均重叠时长: %v\n", avgDuration)
		fmt.Println()
	}
}

// 辅助包级别函数
func PrintConcurrencyAnalysis(collector *CallStackCollector) {
	analyzer := NewOverlapAnalysis(collector)
	analyzer.PrintOverlapReport()
}

// 检测特定函数的并发情况
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

// 获取时间线可视化数据
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

		var nodes []*FunctionCallNode
		oa.collectNodes(tree, &nodes)

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

	// 按开始时间排序
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].StartTime < timeline[j].StartTime
	})

	return timeline
}
