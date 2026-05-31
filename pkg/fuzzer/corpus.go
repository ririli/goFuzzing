package fuzzer

import (
	"fmt"
	"sync"
	"sync/atomic"
	"toolkit/pkg/feedback"
)

// pairKey 生成并发对的唯一键（基于 FuncID 和 CallLoc）
// 因为并发对是动态的，同一对函数可能在不同调用位置出现，需要区分
func pairKey(pair *feedback.SuspiciousPairInfo) string {
	// 确保小的 ID 在前，避免 (1,2) 和 (2,1) 被认为是不同的对
	var funcID1, funcID2 uint64
	var loc1, loc2 feedback.CallLocationInfo

	if pair.FuncID1 <= pair.FuncID2 {
		funcID1, funcID2 = pair.FuncID1, pair.FuncID2
		loc1, loc2 = pair.CallLoc1, pair.CallLoc2
	} else {
		funcID1, funcID2 = pair.FuncID2, pair.FuncID1
		loc1, loc2 = pair.CallLoc2, pair.CallLoc1
	}

	// 包含调用位置信息：文件路径和行号
	return fmt.Sprintf("%d-%d|%s:%d-%s:%d",
		funcID1, funcID2,
		loc1.File, loc1.Line,
		loc2.File, loc2.Line)
}

type CorpusPair struct {
	mu              sync.RWMutex
	isReverse       bool                                    // 是否反转pair
	cnt             uint32                                  // 运行次数
	done            uint32                                  // 是否是第一次
	CoveredConPairs map[string]*feedback.SuspiciousPairInfo // 已覆盖的并发对 (key -> pair)
	SusConPairs     map[string]*feedback.SuspiciousPairInfo // 可疑的并发对 (key -> pair)
	TryPairs        map[string]*feedback.SuspiciousPairInfo // 上次fuzzing输入的并发对 (key -> pair)
	FeedbackPair    map[string]*feedback.SuspiciousPairInfo // fuzzing结束反馈的并发对 (key -> pair)
	// 稳定性追踪：连续 N 轮 pair 集合不变后停止记录调用栈
	stableCount     uint32 // 连续未变化轮次
	stableThreshold uint32 // 稳定判定阈值
	isStable        bool   // 当前是否稳定
	prevPairCount   int    // 上一轮的总 pair 数
}

// Get 获取 TryPairs
func (p *CorpusPair) Get() *feedback.InputPair {
	if atomic.LoadUint32(&p.done) == uint32(0) {
		return nil
	}
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 将 TryPairs 中的 map 转换为 slice
	result := make([]*feedback.SuspiciousPairInfo, 0, len(p.TryPairs))
	for _, pair := range p.TryPairs {
		result = append(result, pair)
	}
	return &feedback.InputPair{
		TryPair:     result,
		RecordStack: !p.isStable,
	}
}

// NewCorpusPair 初始化CorpusPair
func NewCorpusPair() *CorpusPair {
	p := CorpusPair{}
	p.cnt = 0
	p.done = 0
	p.isReverse = true
	p.stableThreshold = 2
	p.CoveredConPairs = make(map[string]*feedback.SuspiciousPairInfo)
	p.SusConPairs = make(map[string]*feedback.SuspiciousPairInfo)
	p.TryPairs = make(map[string]*feedback.SuspiciousPairInfo)
	p.FeedbackPair = make(map[string]*feedback.SuspiciousPairInfo)
	return &p
}

// AddPair 添加并发对，按可信度分类并自动去重
// 1. 将 feedPair 分为 covered 和 suspicious
// 2. 找出 covered 和 TryPairs 的交集 a
// 3. 从 TryPairs 和 SusConPairs 中删除交集 a
// 4. 将 covered 加入 CoveredConPairs
func (p *CorpusPair) AddPair(feedPair []*feedback.SuspiciousPairInfo) {
	defer func() {
		atomic.AddUint32(&p.done, 1)
	}()
	// 第一步：区分 covered 和 suspicious
	covered := make(map[string]*feedback.SuspiciousPairInfo)
	suspicious := make(map[string]*feedback.SuspiciousPairInfo)

	for _, pair := range feedPair {
		if pair == nil {
			continue
		}
		key := pairKey(pair)
		// 严格模式：只有 IsObserved == true 才是 covered
		if pair.IsObserved {
			covered[key] = pair
		} else {
			suspicious[key] = pair
		}
	}

	// 第二步：找出 covered 和 TryPairs 的交集 a
	intersection := make(map[string]struct{})
	for key := range covered {
		if _, ok := p.TryPairs[key]; ok {
			intersection[key] = struct{}{}
		}
	}

	// 第三步：将 covered 加入 CoveredConPairs（covered 中的 IsObserved 都是 true）
	for key, pair := range covered {
		// 直接添加或替换，因为 covered 中的数据都是观测到的（最可靠）
		p.CoveredConPairs[key] = pair
	}

	// 第四步：将 suspicious 加入 SusConPairs（只在不存在时添加）
	for key, pair := range suspicious {
		if _, ok := p.SusConPairs[key]; !ok {
			p.SusConPairs[key] = pair
		}
	}

	// 第五步：从 TryPairs 中删除交集 a
	for key := range intersection {
		delete(p.TryPairs, key)
	}

	// 第六步：从 SusConPairs 中删除交集 a
	for key := range intersection {
		delete(p.SusConPairs, key)
	}

	// 第七步：稳定性判定
	currentCount := len(p.SusConPairs) + len(p.CoveredConPairs)
	if currentCount == p.prevPairCount && currentCount > 0 {
		p.stableCount++
	} else {
		p.stableCount = 0
		p.isStable = false
	}
	p.prevPairCount = currentCount
	if p.stableCount >= p.stableThreshold {
		p.isStable = true
	}

	// todo：运行20次直接稳定
	if atomic.LoadUint32(&p.done) > 20 {
		p.isStable = true
	}

	p.UpdateTryPairs()
}

// UpdateTryPairs 从 SusConPairs 中增量添加最多5个可疑并发对到 TryPairs
// 不会清空 TryPairs，只添加不存在的新项
func (p *CorpusPair) UpdateTryPairs() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cnt++
	// 从 SusConPairs 中增量添加最多5个到 TryPairs
	count := 0
	for key, pair := range p.SusConPairs {
		if count >= 5 {
			break
		}
		// 只添加 TryPairs 中不存在的
		if _, ok := p.TryPairs[key]; !ok {
			p.TryPairs[key] = pair
			count++
		}
	}
	if p.isReverse && p.cnt%2 == 0 {
		// 反转 TryPairs 中所有 pair 的 FuncID 和 CallLoc
		// 当 pre 函数结束过快导致没有并发执行时，通过反转顺序
		// 让原本的 next 先执行、原本的 pre 后执行，交替尝试创造时间重叠
		for _, pair := range p.TryPairs {
			pair.FuncID1, pair.FuncID2 = pair.FuncID2, pair.FuncID1
			pair.CallLoc1, pair.CallLoc2 = pair.CallLoc2, pair.CallLoc1
		}
	}
}
