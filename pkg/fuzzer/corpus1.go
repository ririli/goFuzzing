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
	mu              sync.Mutex
	done            uint32                                  // 是否是第一次
	CoveredConPairs map[string]*feedback.SuspiciousPairInfo // 已覆盖的并发对 (key -> pair)
	SusConPairs     map[string]*feedback.SuspiciousPairInfo // 可疑的并发对 (key -> pair)
	TryPairs        map[string]*feedback.SuspiciousPairInfo // 上次fuzzing输入的并发对 (key -> pair)
	FeedbackPair    map[string]*feedback.SuspiciousPairInfo // fuzzing结束反馈的并发对 (key -> pair)
}

// Get 获取 TryPairs
func (p *CorpusPair) Get() *feedback.InputPair {
	if atomic.LoadUint32(&p.done) == uint32(0) {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	// 将 TryPairs 中的 map 转换为 slice
	result := make([]*feedback.SuspiciousPairInfo, 0, len(p.TryPairs))
	for _, pair := range p.TryPairs {
		result = append(result, pair)
	}
	return &feedback.InputPair{
		TryPair: result,
	}
}

// NewCorpusPair 初始化CorpusPair
func NewCorpusPair() *CorpusPair {
	p := CorpusPair{}
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

	p.UpdateTryPairs()
}

// UpdateTryPairs 从 SusConPairs 中增量添加最多5个可疑并发对到 TryPairs
// 不会清空 TryPairs，只添加不存在的新项
func (p *CorpusPair) UpdateTryPairs() {
	p.mu.Lock()
	defer p.mu.Unlock()

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
}
