package callstack

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	collector = NewCallStackCollector() // 收集的全部函数调用链
	mu        sync.Mutex

	waiters sync.Map
)
var (
	cfg           *Config
	timeout       time.Duration
	timeoutGlobal time.Duration
	oa            OverlapAnalysis
)

func init() {
	cfg = NewConfig()
	timeout = 500 * time.Millisecond

}

// ParseInput 解析输入
func ParseInput() {
	input_susPairs := os.Getenv("Input")
	if input_susPairs != "" {
		ParseSusPairs(input_susPairs)
	}
}

// ParseSusPairs 解析输入的函数对
// 格式: (id1,id2)(id3,id4)(id5,id6)...
func ParseSusPairs(s string) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	// 逐对解析 (id1,id2) 格式
	for len(s) > 0 {
		// 查找左括号
		left := strings.Index(s, "(")
		if left == -1 {
			break
		}
		// 查找右括号
		right := strings.Index(s[left:], ")")
		if right == -1 {
			break
		}
		right += left // 调整为绝对位置

		// 提取括号内的内容
		pairStr := s[left+1 : right]

		// 解析两个 ID
		var id1, id2 uint64
		_, err := fmt.Sscanf(pairStr, "%d,%d", &id1, &id2)
		if err == nil {
			// TODO: 处理提取出的 id1 和 id2
			// 这里可以调用后续的处理函数
		}

		// 移动到下一对
		s = s[right+1:]
	}
}

// PrintConPairs 打印所有并发函数对到stderr
// 格式：[CONPAIR] node1:funcId = xxx,callloc = xxx;node2:funcid = xxx,callloc = xxx;
func PrintSusConPairs() {
	// 重新检测并发函数对（在测试结束时调用，此时所有函数都已执行完毕）
	pairs := oa.DetectFunctionOverlaps()

	if len(pairs) == 0 {
		print("No concurrent function pairs found.\n")
		return
	} else {
		print("Concurrent function pairs found:\n")
	}

	susPairs := InferSuspiciousPairs(pairs)

	for _, pair := range susPairs {
		info := pair.String()
		print(info)
	}
}

// Trace 自动插桩函数，在函数开始处调用 用法：defer Trace(funcID)()
func Trace(funcID uint64) func() {

	pointControl(funcID)

	mu.Lock()
	node := collector.EnterFunction(funcID)
	mu.Unlock()

	// 返回的闭包将在defer时执行
	return func() {
		mu.Lock()
		defer mu.Unlock()
		collector.ExitFunction(node)
	}
}

// findPrev 查找指定 funcId 的前驱 ID
func (c *Config) findPrev(funcId uint64) []uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if prevId, ok := c.preFuncMap[funcId]; ok {
		return prevId
	}
	return nil // 没有前驱
}

// todo
// pointControl 实现函数对之间的断点控制
func pointControl(funcId uint64) {

	if !cfg.isActive(funcId) {
		return
	}
	//判断函数是否需要等待
	if cfg.doWait(funcId) {
		preIds := cfg.findPrev(funcId)
		if preIds != nil {
			for _, preId := range preIds {
				waiter := getWaiter(preId)

				select {
				case <-waiter:
					cfg.waitMapDec(funcId)
					fmt.Printf("[COVERED] {%v, %v}\n", preId, funcId)
				case <-time.After(timeout):
					fmt.Printf("[TIMEOUT] {%v, %v}\n", preId, funcId)
				}
			}
		}
	}
	completeOperation(funcId)
}

// isActive 判断函数是否处于活动状态
func (c *Config) isActive(funcId uint64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.activeFunc[funcId]
	return ok
}

// doWait 判断函数是否需要等待
func (c *Config) doWait(funcId uint64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if value, ok := c.waitMap[funcId]; ok {
		return value > 0
	}
	return false
}

// waitMapDec 减少指定函数 ID 的等待计数
func (c *Config) waitMapDec(funcId uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if v, ok := c.waitMap[funcId]; ok {
		if v <= 1 {
			delete(c.waitMap, funcId)
		} else {
			c.waitMap[funcId]--
		}
	}
}

// getWaiter 获取或创建指定操作 ID 的等待 channel
func getWaiter(id uint64) chan struct{} {
	// 尝试加载已存在的 waiter
	if val, ok := waiters.Load(id); ok {
		return val.(chan struct{})
	}

	// 创建新的 waiter
	newWaiter := make(chan struct{}, 1) // 缓冲为 1，避免发送时阻塞

	// 存储，如果已被其他协程创建则使用已有的
	actual, _ := waiters.LoadOrStore(id, newWaiter)
	return actual.(chan struct{})
}

// completeOperation 标记操作完成，通知所有等待者
func completeOperation(id uint64) {
	if val, ok := waiters.Load(id); ok {
		ch := val.(chan struct{})
		close(ch) // 关闭 channel，所有等待者都会收到信号
	} else {
		done := make(chan struct{})
		close(done)
		waiters.LoadOrStore(id, done)
	}
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
