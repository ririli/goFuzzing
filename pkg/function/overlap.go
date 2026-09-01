package function

import (
	"sync"
)

// CallLocation 记录函数调用位置信息
type CallLocation struct {
	File string // 源文件路径
	Line int    // 行号
}

// FuncRecord 记录单个函数调用实例的生命周期
type FuncRecord struct {
	FuncID    uint64
	CallID    uint64
	ParentID  uint64
	StartTime int64 // UnixNano
	EndTime   int64 // UnixNano, 0 = 仍在运行
	CallLoc   CallLocation
}

// funcEdge 表示一次执行中聚合后的函数调用者-被调用者边。
type funcEdge struct {
	Caller uint64 `json:"caller"`
	Callee uint64 `json:"callee"`
	Count  uint64 `json:"count"`
}

// FuncTracker 收集函数调用生命周期记录，对标 GoroutineTracker
type FuncTracker struct {
	mu         sync.RWMutex
	funcMap    map[uint64][]*FuncRecord // funcID -> 实例列表
	callerMap  map[uint64]uint64        // callee funcID -> caller funcID（最近一次）
	nextCallID uint64
	depth      map[int][]*FuncRecord // OS goroutine ID -> 调用栈（用于父函数追踪）
}

// NewFuncTracker 创建新的函数调用追踪器
func NewFuncTracker() *FuncTracker {
	return &FuncTracker{
		funcMap:   make(map[uint64][]*FuncRecord),
		callerMap: make(map[uint64]uint64),
		depth:     make(map[int][]*FuncRecord),
	}
}

// snapshotEdges 返回当前执行中去重并计数的调用者-被调用者边快照。
func (ft *FuncTracker) snapshotEdges() []funcEdge {
	ft.mu.RLock()
	defer ft.mu.RUnlock()

	type edgeKey struct {
		caller uint64
		callee uint64
	}
	counts := make(map[edgeKey]uint64)
	for callee, caller := range ft.callerMap {
		if callee == 0 || caller == callee {
			continue
		}
		counts[edgeKey{caller: caller, callee: callee}]++
	}

	edges := make([]funcEdge, 0, len(counts))
	for key, count := range counts {
		edges = append(edges, funcEdge{
			Caller: key.caller,
			Callee: key.callee,
			Count:  count,
		})
	}
	sortFuncEdges(edges)
	return edges
}
