// Package overlap 提供函数执行时间重叠检测与可疑并发对推断。
// 它依赖 calltree 包的调用树数据模型。
package overlap

import (
	"fmt"

	"toolkit/pkg/calltree"
)

// TimeOverlap 表示两个函数执行时重叠的时间区间
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
	Node1   *calltree.FunctionCallNode
	Node2   *calltree.FunctionCallNode
	Overlap TimeOverlap
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
