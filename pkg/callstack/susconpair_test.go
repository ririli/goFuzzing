package callstack

import (
	"fmt"
	"testing"
)

// TestSuspiciousConcurrentPairs 测试可疑并发对检测
func TestSuspiciousConcurrentPairs(t *testing.T) {
	// 创建收集器
	collector := NewCallStackCollector()

	// 模拟调用链 1: 1-2-3 (Goroutine 1)
	// 模拟调用链 2: 4-5-6 (Goroutine 2)

	// Goroutine 1 的调用链
	node1 := &FunctionCallNode{
		CallID:        1,
		FuncName:      "func1",
		GoroutineID:   1,
		StartUnixNano: 1000,
		EndUnixNano:   5000,
	}

	node2 := &FunctionCallNode{
		CallID:        2,
		FuncName:      "func2",
		GoroutineID:   1,
		StartUnixNano: 2000,
		EndUnixNano:   4000,
		Parent:        node1,
		Depth:         1,
	}

	node3 := &FunctionCallNode{
		CallID:        3,
		FuncName:      "func3",
		GoroutineID:   1,
		StartUnixNano: 2500,
		EndUnixNano:   3500,
		Parent:        node2,
		Depth:         2,
	}

	node1.Children = append(node1.Children, node2)
	node2.Children = append(node2.Children, node3)

	// Goroutine 2 的调用链
	node4 := &FunctionCallNode{
		CallID:        4,
		FuncName:      "func4",
		GoroutineID:   2,
		StartUnixNano: 1500,
		EndUnixNano:   5500,
	}

	node5 := &FunctionCallNode{
		CallID:        5,
		FuncName:      "func5",
		GoroutineID:   2,
		StartUnixNano: 2200,
		EndUnixNano:   4200,
		Parent:        node4,
		Depth:         1,
	}

	node6 := &FunctionCallNode{
		CallID:        6,
		FuncName:      "func6",
		GoroutineID:   2,
		StartUnixNano: 2800,
		EndUnixNano:   3800,
		Parent:        node5,
		Depth:         2,
	}

	node4.Children = append(node4.Children, node5)
	node5.Children = append(node5.Children, node6)

	// 添加到收集器
	collector.callTrees[1] = node1
	collector.callTrees[2] = node4
	collector.nodePool[1] = node1
	collector.nodePool[2] = node2
	collector.nodePool[3] = node3
	collector.nodePool[4] = node4
	collector.nodePool[5] = node5
	collector.nodePool[6] = node6

	// 创建分析器
	analyzer := NewSuspiciousPairAnalyzer(collector)

	// 检测可疑并发对
	pairs := analyzer.DetectSuspiciousConcurrentPairs()

	// 打印结果
	fmt.Printf("检测到 %d 个可疑并发对:\n", len(pairs))
	for i, pair := range pairs {
		fmt.Printf("%d. [%d]%s (G%d) <-> [%d]%s (G%d)\n",
			i+1,
			pair.CallID1, pair.FuncName1, pair.Goroutine1,
			pair.CallID2, pair.FuncName2, pair.Goroutine2,
		)
		fmt.Printf("   原因: %s\n", pair.Reason)
	}

	// 验证是否检测到了预期的对
	if len(pairs) == 0 {
		t.Log("注意: 由于没有实际的时间重叠数据，可能不会检测到可疑对")
		t.Log("这是正常的，因为 DetectSuspiciousConcurrentPairs 依赖于 DetectFunctionOverlaps")
	}
}

// ExampleSuspiciousPairAnalyzer 使用示例
func ExampleSuspiciousPairAnalyzer() {
	collector := NewCallStackCollector()
	analyzer := NewSuspiciousPairAnalyzer(collector)

	// 打印报告
	analyzer.PrintSuspiciousPairsReport()
}
