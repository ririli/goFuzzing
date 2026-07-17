package goroutine

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// GoroutineRecord 记录单个goroutine实例的生命周期
type GoroutineRecord struct {
	Gid       uint64           // 静态goroutine ID（来自插桩）
	GortID    int              // 运行时goroutine ID（来自runtime.Stack，用于调试）
	ParentGid uint64           // 父goroutine的静态ID（0=主goroutine，或创建此goroutine的goroutine）
	StartTime int64            // UnixNano
	EndTime   int64            // UnixNano（0表示尚未结束）
	CallLoc   CallLocationInfo // go语句的源位置
}

// goroutineEdge 表示一次执行中聚合后的静态goroutine父子边。
type goroutineEdge struct {
	ParentGid uint64 `json:"parent"`
	ChildGid  uint64 `json:"child"`
	Count     uint64 `json:"count"`
}

// GoroutineTracker 收集goroutine生命周期记录
type GoroutineTracker struct {
	mu       sync.RWMutex
	gortMap  map[uint64][]*GoroutineRecord // gid -> 实例列表
	childMap map[uint64][]uint64           // parentGid -> childGid 列表
}

// NewGoroutineTracker 创建新的goroutine追踪器
func NewGoroutineTracker() *GoroutineTracker {
	return &GoroutineTracker{
		gortMap:  make(map[uint64][]*GoroutineRecord),
		childMap: make(map[uint64][]uint64),
	}
}

// EnterGoroutine 记录goroutine开始（无父goroutine信息）
func (gt *GoroutineTracker) EnterGoroutine(gid uint64) {
	gt.EnterGoroutineWithParent(gid, 0)
}

// EnterGoroutineWithParent 记录goroutine开始并附带父goroutine ID
func (gt *GoroutineTracker) EnterGoroutineWithParent(gid uint64, parentGid uint64) {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	rec := &GoroutineRecord{
		Gid:       gid,
		GortID:    getCurrentGoroutineID(),
		ParentGid: parentGid,
		StartTime: time.Now().UnixNano(),
	}
	gt.gortMap[gid] = append(gt.gortMap[gid], rec)
	if gid != parentGid {
		gt.childMap[parentGid] = append(gt.childMap[parentGid], gid)
	}
}

// ExitGoroutine 记录goroutine结束
// 找到最近一个未结束的实例并记录其结束时间
func (gt *GoroutineTracker) ExitGoroutine(gid uint64) {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	instances := gt.gortMap[gid]
	if len(instances) == 0 {
		return
	}
	// 找到最近一个未结束的实例
	for i := len(instances) - 1; i >= 0; i-- {
		if instances[i].EndTime == 0 {
			instances[i].EndTime = time.Now().UnixNano()
			return
		}
	}
}

// EnterGoroutineWithLoc 记录goroutine开始并附带调用位置信息（无父goroutine信息）
func (gt *GoroutineTracker) EnterGoroutineWithLoc(gid uint64, file string, line int) {
	gt.mu.Lock()
	defer gt.mu.Unlock()

	rec := &GoroutineRecord{
		Gid:       gid,
		GortID:    getCurrentGoroutineID(),
		ParentGid: 0,
		StartTime: time.Now().UnixNano(),
		CallLoc: CallLocationInfo{
			File: file,
			Line: line,
		},
	}
	gt.gortMap[gid] = append(gt.gortMap[gid], rec)
}

// getParentGid 从gortMap中获取指定gid的父gid。
// 取第一个实例的ParentGid。假设同一gid的所有实例有相同父节点——
// 在静态gid机制下此假设成立（同一go语句处编译期分配同一gid）。
// 调用方需持有mu读锁。
func (gt *GoroutineTracker) getParentGid(gid uint64) uint64 {
	instances := gt.gortMap[gid]
	if len(instances) == 0 {
		return 0
	}
	return instances[0].ParentGid
}

// snapshotEdges 返回当前执行中去重并计数的父子边快照。
func (gt *GoroutineTracker) snapshotEdges() []goroutineEdge {
	gt.mu.RLock()
	defer gt.mu.RUnlock()

	type edgeKey struct {
		parent uint64
		child  uint64
	}

	counts := make(map[edgeKey]uint64)
	for gid, instances := range gt.gortMap {
		for _, instance := range instances {
			if instance.ParentGid == gid {
				continue
			}
			counts[edgeKey{parent: instance.ParentGid, child: gid}]++
		}
	}

	edges := make([]goroutineEdge, 0, len(counts))
	for key, count := range counts {
		edges = append(edges, goroutineEdge{
			ParentGid: key.parent,
			ChildGid:  key.child,
			Count:     count,
		})
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].ParentGid != edges[j].ParentGid {
			return edges[i].ParentGid < edges[j].ParentGid
		}
		return edges[i].ChildGid < edges[j].ChildGid
	})
	return edges
}

// PrintGoroutineRecords 打印所有goroutine实例的追踪记录，用于调试查看
// 输出格式:
//
//	=== Goroutine Records ===
//	gid=0, instances=1, children=[10,20]
//	  [0] gortID=1, parent=0, start=100, end=900 (dur=800ns)
//	gid=10, instances=2, children=[20]
//	  [0] gortID=22, parent=0, start=200, end=400 (dur=200ns)
//	  [1] gortID=24, parent=0, start=500, end=800 (dur=300ns)
func (gt *GoroutineTracker) PrintGoroutineRecords() {
	gt.mu.RLock()
	defer gt.mu.RUnlock()

	fmt.Println("=== Goroutine Records ===")

	// 收集所有 gid 并排序
	gids := make([]uint64, 0, len(gt.gortMap))
	for gid := range gt.gortMap {
		gids = append(gids, gid)
	}
	sort.Slice(gids, func(i, j int) bool { return gids[i] < gids[j] })

	for _, gid := range gids {
		instances := gt.gortMap[gid]
		children := gt.childMap[gid]
		fmt.Printf("gid=%d, instances=%d, children=%v\n", gid, len(instances), children)

		for i, inst := range instances {
			dur := int64(0)
			if inst.EndTime != 0 {
				dur = inst.EndTime - inst.StartTime
			}
			endStr := fmt.Sprintf("%d", inst.EndTime)
			if inst.EndTime == 0 {
				endStr = "(running)"
			}
			fmt.Printf("  [%d] gortID=%d, parent=%d, start=%d, end=%s (dur=%dns)\n",
				i, inst.GortID, inst.ParentGid, inst.StartTime, endStr, dur)
		}
	}
}
