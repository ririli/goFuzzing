package goroutine

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// CallLocationInfo 表示goroutine创建位置信息
type CallLocationInfo struct {
	File string // 源文件路径
	Line int    // 行号
}

// GoroutineRecord 记录单个goroutine实例的生命周期
type GoroutineRecord struct {
	Gid       uint64           // 静态goroutine ID（来自插桩）
	GortID    int              // 运行时goroutine ID（来自runtime.Stack，用于调试）
	ParentGid uint64           // 父goroutine的静态ID（0=主goroutine，或创建此goroutine的goroutine）
	StartTime int64            // UnixNano
	EndTime   int64            // UnixNano（0表示尚未结束）
	CallLoc   CallLocationInfo // go语句的源位置
}

// GoroutineTracker 收集goroutine生命周期记录
type GoroutineTracker struct {
	mu      sync.RWMutex
	gortMap map[uint64][]*GoroutineRecord // gid -> 实例列表
}

// NewGoroutineTracker 创建新的goroutine追踪器
func NewGoroutineTracker() *GoroutineTracker {
	return &GoroutineTracker{
		gortMap: make(map[uint64][]*GoroutineRecord),
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

// GoroutinePairInfo 表示一对并发goroutine
type GoroutinePairInfo struct {
	Gid1       uint64           // 第一个goroutine的ID
	Gid2       uint64           // 第二个goroutine的ID
	CallLoc1   CallLocationInfo // 第一个go语句的位置
	CallLoc2   CallLocationInfo // 第二个go语句的位置
	Confidence float64          // 置信度（直接观测始终为1.0）
	SourceType string           // 来源类型（始终为"observed"）
}

// String 返回字符串表示（格式与callstack包一致，便于fuzzer解析）
// 置信度1.0输出[COVERED]，否则输出[SUSPECT]
func (g *GoroutinePairInfo) String() string {
	if g.Confidence >= 1.0 {
		return fmt.Sprintf("[COVERED] %d,%d|%s:%d,%s:%d|%.2f|%s;\n",
			g.Gid1, g.Gid2,
			g.CallLoc1.File, g.CallLoc1.Line,
			g.CallLoc2.File, g.CallLoc2.Line,
			g.Confidence, g.SourceType)
	}
	return fmt.Sprintf("[SUSPECT] %d,%d|%s:%d,%s:%d|%.2f|%s;\n",
		g.Gid1, g.Gid2,
		g.CallLoc1.File, g.CallLoc1.Line,
		g.CallLoc2.File, g.CallLoc2.Line,
		g.Confidence, g.SourceType)
}

// DetectGoroutineOverlaps 检测所有goroutine ID之间的时间重叠
// 对于每对goroutine ID，检查是否有任何实例存在时间重叠
func (gt *GoroutineTracker) DetectGoroutineOverlaps() []*GoroutinePairInfo {
	gt.mu.RLock()
	defer gt.mu.RUnlock()

	// 1. 计算每个gid的时间范围: [min(StartTime), max(EndTime)]
	type gidRange struct {
		minStart int64
		maxEnd   int64
		callLoc  CallLocationInfo
	}

	ranges := make(map[uint64]*gidRange)
	for gid, instances := range gt.gortMap {
		var minStart, maxEnd int64
		first := true
		var loc CallLocationInfo
		for _, inst := range instances {
			if inst.EndTime == 0 {
				continue // 跳过仍在运行的实例
			}
			if first {
				minStart = inst.StartTime
				maxEnd = inst.EndTime
				loc = inst.CallLoc
				first = false
			} else {
				if inst.StartTime < minStart {
					minStart = inst.StartTime
				}
				if inst.EndTime > maxEnd {
					maxEnd = inst.EndTime
				}
			}
		}
		if !first {
			ranges[gid] = &gidRange{
				minStart: minStart,
				maxEnd:   maxEnd,
				callLoc:  loc,
			}
		}
	}

	// 2. O(n²) 两两比较
	gids := make([]uint64, 0, len(ranges))
	for gid := range ranges {
		gids = append(gids, gid)
	}
	sort.Slice(gids, func(i, j int) bool { return gids[i] < gids[j] })

	var overlaps []*GoroutinePairInfo
	for i := 0; i < len(gids); i++ {
		for j := i + 1; j < len(gids); j++ {
			gid1, gid2 := gids[i], gids[j]
			r1, r2 := ranges[gid1], ranges[gid2]

			if isTimeRangeOverlap(r1.minStart, r1.maxEnd, r2.minStart, r2.maxEnd) {
				overlaps = append(overlaps, &GoroutinePairInfo{
					Gid1:       gid1,
					Gid2:       gid2,
					CallLoc1:   r1.callLoc,
					CallLoc2:   r2.callLoc,
					Confidence: 1.0,
					SourceType: "observed",
				})
			}
		}
	}

	// 按重叠置信度保持兼容
	return overlaps
}

// InferParentChildPairs 推测父子goroutine并发对 (Rule 2, confidence=0.9)
// 对于每个有父goroutine的记录，生成 (ParentGid, Gid) 对
func (gt *GoroutineTracker) InferParentChildPairs() []*GoroutinePairInfo {
	gt.mu.RLock()
	defer gt.mu.RUnlock()

	var pairs []*GoroutinePairInfo
	seen := make(map[string]bool)

	for gid, instances := range gt.gortMap {
		for _, inst := range instances {
			if inst.ParentGid == 0 || inst.ParentGid == gid {
				continue
			}
			// 生成标准化的key避免重复
			gid1, gid2 := inst.ParentGid, gid
			if gid1 > gid2 {
				gid1, gid2 = gid2, gid1
			}
			key := fmt.Sprintf("%d-%d", gid1, gid2)
			if seen[key] {
				continue
			}
			seen[key] = true

			pairs = append(pairs, &GoroutinePairInfo{
				Gid1:       inst.ParentGid,
				Gid2:       gid,
				Confidence: 0.9,
				SourceType: "inferred_parent_child",
			})
		}
	}
	return pairs
}

// InferSelfPairs 推测自配对并发对 (Rule 3, confidence=0.7)
// 对于有多个实例的goroutine ID，生成 (Gid, Gid) 自配对
// 表示同一goroutine ID的多个实例之间可能存在并发
func (gt *GoroutineTracker) InferSelfPairs() []*GoroutinePairInfo {
	gt.mu.RLock()
	defer gt.mu.RUnlock()

	var pairs []*GoroutinePairInfo

	for gid, instances := range gt.gortMap {
		// 统计已完成的实例数
		completed := 0
		for _, inst := range instances {
			if inst.EndTime != 0 {
				completed++
			}
		}
		if completed >= 2 {
			// 至少有两个实例完成了 → 存在自配对可能性
			pairs = append(pairs, &GoroutinePairInfo{
				Gid1:       gid,
				Gid2:       gid,
				Confidence: 0.7,
				SourceType: "inferred_self_pair",
			})
		}
	}
	return pairs
}

// isTimeRangeOverlap 检测两个时间区间是否重叠
// 使用 <= 确保零时长（start==end）的瞬时调用也能被检测到并发
func isTimeRangeOverlap(start1, end1, start2, end2 int64) bool {
	if end1 == 0 || end2 == 0 {
		return false
	}
	return start1 <= end2 && start2 <= end1
}
