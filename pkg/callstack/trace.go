package callstack

import (
	"fmt"
	"sync"
)

var (
	collector = NewCallStackCollector()
	mu        sync.Mutex
)

// Trace 自动插桩函数，在函数开始处调用
// 用法：defer Trace()()
func Trace() func() {
	mu.Lock()
	defer mu.Unlock()

	node := collector.EnterFunction()

	// 返回的闭包将在defer时执行
	return func() {
		mu.Lock()
		defer mu.Unlock()
		collector.ExitFunction(node)
	}
}

//// TraceWithName 手动指定函数名（用于特殊场景）
//func TraceWithName(funcName string) func() {
//	mu.Lock()
//	defer mu.Unlock()
//
//	// 这里可以自定义节点创建逻辑
//	goroutineID := getCurrentGoroutineID()
//
//	node := &callstack.FunctionCallNode{
//		CallID:      generateCallID(), // 需要实现
//		FuncName:    funcName,
//		GoroutineID: goroutineID,
//		StartTime:   time.Now(),
//	}
//
//	// 将节点添加到收集器（需要扩展collector接口）
//
//	return func() {
//		mu.Lock()
//		defer mu.Unlock()
//		collector.ExitFunction(node)
//	}
//}

// GetCollector 获取收集器实例
func GetCollector() *CallStackCollector {
	return collector
}

func PrintTrees() {
	// 获取并打印调用树
	trees := collector.GetAllCallTrees()

	for goroutineID, _ := range trees {
		fmt.Printf("\n")
		collector.PrintCallTree(goroutineID)

		// 获取统计信息
		stats := collector.GetStatistics(goroutineID)
		if stats != nil {
			fmt.Printf("\n统计信息:\n")
			fmt.Printf("总调用次数: %d\n", stats.TotalCalls)
			fmt.Printf("总执行时间: %v\n", stats.TotalTime)
			fmt.Printf("平均调用时间: %v\n", stats.AvgTime)
		}
	}
}
