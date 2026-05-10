package callstack

import (
	"fmt"
	"sync"
)

type Config struct {
	mu sync.RWMutex
	//funcPair          []ConPairFunc              // 当前运行收集的并发调用对
	suspiciousConPair []SuspiciousConcurrentPair //推测出可能的调用对，后续用断点控制进行验证
	preFuncMap        map[uint64][]uint64        // 记录两个函数的前驱
	activeFunc        map[uint64]struct{}        // 需要验证的函数
	waitMap           map[uint64]int32           // 记录需要等待的函数,value 表示等待的preFunc数量

}

func NewConfig() *Config {
	cfg := Config{}
	//cfg.funcPair = make([]ConPairFunc, 0)
	cfg.suspiciousConPair = make([]SuspiciousConcurrentPair, 0)
	cfg.preFuncMap = make(map[uint64][]uint64)
	cfg.activeFunc = make(map[uint64]struct{})
	cfg.waitMap = make(map[uint64]int32)
	return &cfg
}

// LoadSusPairs 加载一组可疑的并发对
func (c *Config) LoadSusPairs(conPairs []ConPairFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, pair := range conPairs {
		// 获取两个重叠节点
		node1 := collector.GetNodeByID(pair.CallID1)
		node2 := collector.GetNodeByID(pair.CallID2)

		if node1 == nil || node2 == nil {
			continue
		}

		// 推断规则1: node1的父节点 × node2本身
		if node1.Parent != nil {
			susPair := SuspiciousConcurrentPair{
				FuncID1:    node1.Parent.FuncID,
				FuncID2:    node2.FuncID,
				CallID1:    node1.Parent.CallID,
				CallID2:    node2.CallID,
				FuncName1:  node1.Parent.FuncName,
				FuncName2:  node2.FuncName,
				Goroutine1: node1.Parent.GoroutineID,
				Goroutine2: node2.GoroutineID,
				Reason:     fmt.Sprintf("基于重叠对 (%s@%d, %s@%d) 推断: node1父节点×node2", pair.Func1, pair.CallID1, pair.Func2, pair.CallID2),
			}
			c.suspiciousConPair = append(c.suspiciousConPair, susPair)
		}

		// 推断规则2: node1的子节点 × node2本身
		for _, child1 := range node1.Children {
			susPair := SuspiciousConcurrentPair{
				FuncID1:    child1.FuncID,
				FuncID2:    node2.FuncID,
				CallID1:    child1.CallID,
				CallID2:    node2.CallID,
				FuncName1:  child1.FuncName,
				FuncName2:  node2.FuncName,
				Goroutine1: child1.GoroutineID,
				Goroutine2: node2.GoroutineID,
				Reason:     fmt.Sprintf("基于重叠对 (%s@%d, %s@%d) 推断: node1子节点×node2", pair.Func1, pair.CallID1, pair.Func2, pair.CallID2),
			}
			c.suspiciousConPair = append(c.suspiciousConPair, susPair)
		}

		// 推断规则3: node1本身 × node2的父节点
		if node2.Parent != nil {
			susPair := SuspiciousConcurrentPair{
				FuncID1:    node1.FuncID,
				FuncID2:    node2.Parent.FuncID,
				CallID1:    node1.CallID,
				CallID2:    node2.Parent.CallID,
				FuncName1:  node1.FuncName,
				FuncName2:  node2.Parent.FuncName,
				Goroutine1: node1.GoroutineID,
				Goroutine2: node2.Parent.GoroutineID,
				Reason:     fmt.Sprintf("基于重叠对 (%s@%d, %s@%d) 推断: node1×node2父节点", pair.Func1, pair.CallID1, pair.Func2, pair.CallID2),
			}
			c.suspiciousConPair = append(c.suspiciousConPair, susPair)
		}

		// 推断规则4: node1本身 × node2的子节点
		for _, child2 := range node2.Children {
			susPair := SuspiciousConcurrentPair{
				FuncID1:    node1.FuncID,
				FuncID2:    child2.FuncID,
				CallID1:    node1.CallID,
				CallID2:    child2.CallID,
				FuncName1:  node1.FuncName,
				FuncName2:  child2.FuncName,
				Goroutine1: node1.GoroutineID,
				Goroutine2: child2.GoroutineID,
				Reason:     fmt.Sprintf("基于重叠对 (%s@%d, %s@%d) 推断: node1×node2子节点", pair.Func1, pair.CallID1, pair.Func2, pair.CallID2),
			}
			c.suspiciousConPair = append(c.suspiciousConPair, susPair)
		}
	}
}

func (c *Config) LoadInfo() {

	for _, pair := range c.suspiciousConPair {
		// 记录前驱关系：pair.FuncID2 的前驱是 pair.FuncID1
		// 注意：如果一个函数有多个前驱，这里可能需要用 map[uint64][]uint64
		c.preFuncMap[pair.FuncID2] = append(c.preFuncMap[pair.FuncID2], pair.FuncID1)

		c.waitMap[pair.FuncID2]++ // 标记需要等待
		c.activeFunc[pair.FuncID1] = struct{}{}
		c.activeFunc[pair.FuncID2] = struct{}{}
	}
}
