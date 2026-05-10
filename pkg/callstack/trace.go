package callstack

import (
	"fmt"
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
	conPairs      []ConPairFunc
)

func init() {
	cfg = NewConfig()
	timeout = 500 * time.Millisecond
	oa.collector = collector
	conPairs = oa.DetectFunctionOverlaps()
	cfg.LoadSusPairs(conPairs)
	cfg.LoadInfo()
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
