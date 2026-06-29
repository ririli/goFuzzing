// Package callstack 是运行时并发测试插桩的门面包。
//
// 它组装 calltree（调用树收集）、overlap（时间重叠检测）、breakpoint（断点控制）
// 三个子包，对外提供稳定的 Trace / ParseInput / PrintSusConPairs / PrintTrees API。
//
// 本包是被插桩代码的唯一入口 —— inst/passes 在源码中注入的 defer callstack.Trace(id)()
// 和 callstack.ParseInput() 等调用全部指向本包。
//
// 断点策略切换：
//
//	默认使用单栏策略（Config），设置环境变量 BARRIER_MODE=double 可切换为双栏策略（BarrierConfig）。
package callstack

import (
	"fmt"
	"os"
	"time"

	"toolkit/pkg/breakpoint"
	"toolkit/pkg/calltree"
	"toolkit/pkg/overlap"
)

var (
	collector *calltree.CallStackCollector // 全局调用树收集器
	strategy  breakpoint.Strategy          // 当前断点控制策略（单栏或双栏）
	oa        *overlap.OverlapAnalysis     // 全局重叠分析器
)

func init() {
	collector = calltree.NewCallStackCollector()
	oa = overlap.NewOverlapAnalysis(collector)

	// 根据环境变量选择断点策略
	if os.Getenv("BARRIER_MODE") == "double" {
		strategy = breakpoint.NewBarrierConfig()
	} else {
		strategy = breakpoint.NewConfig()
	}
}

// Trace 自动插桩函数，用法：defer Trace(funcID)()
//
// 它依次执行：
//  1. 断点控制（根据当前策略决定是否阻塞等待）
//  2. 调用树记录（除非 RECORD_STACK=1 跳过）
func Trace(funcID uint64) func() {
	strategy.PointControl(funcID)

	if os.Getenv("RECORD_STACK") == "1" {
		return func() {}
	}

	node := collector.EnterFunction(funcID)
	return func() {
		collector.ExitFunction(node)
	}
}

// ParseInput 从环境变量 Input 解析可疑函数对配置
func ParseInput() {
	strategy.ParseInput()
}

// PrintSusConPairs 检测并打印所有可疑的并发函数对到 stderr
func PrintSusConPairs() {
	if os.Getenv("RECORD_STACK") == "1" {
		return
	}
	time.Sleep(500 * time.Millisecond) // 等待子goroutine执行完毕

	pairs := oa.DetectFunctionOverlaps()
	if len(pairs) == 0 {
		print("No concurrent function pairs found.\n")
		return
	}
	print("Concurrent function pairs found:\n")

	susPairs := overlap.InferSuspiciousPairs(pairs)
	for _, pair := range susPairs {
		print(pair.String())
	}
}

// PrintTrees 打印所有 goroutine 的调用树和统计信息
func PrintTrees() {
	trees := collector.GetAllCallTrees()
	for goroutineID := range trees {
		fmt.Printf("\n")
		collector.PrintCallTree(goroutineID)

		stats := collector.GetStatistics(goroutineID)
		if stats != nil {
			fmt.Printf("\n统计信息:\n")
			fmt.Printf("总调用次数: %d\n", stats.TotalCalls)
			fmt.Printf("总执行时间: %v\n", stats.TotalTime)
			fmt.Printf("平均调用时间: %v\n", stats.AvgTime)
		}
	}
}
