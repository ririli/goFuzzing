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

// signalKey 生成调度信号的查找键（仅基于 FuncID，不含 CallLoc）
func signalKey(preID, nextID uint64) string {
	if preID <= nextID {
		return fmt.Sprintf("%d-%d", preID, nextID)
	}
	return fmt.Sprintf("%d-%d", nextID, preID)
}

type CoolDownState int

const (
	StateHot  CoolDownState = iota // 每轮收集
	StateWarm                      // 每 2 轮收集一次
	StateCool                      // 每 4 轮收集一次
	StateCold                      // 每 8 轮收集一次，永不彻底停止
)

func (s CoolDownState) String() string {
	switch s {
	case StateHot:
		return "HOT"
	case StateWarm:
		return "WARM"
	case StateCool:
		return "COOL"
	case StateCold:
		return "COLD"
	}
	return "UNKNOWN"
}

type CorpusPair struct {
	mu              sync.RWMutex
	isReverse       bool                                    // 是否反转pair
	done            uint32                                  // 是否是第一次
	CoveredConPairs map[string]*feedback.SuspiciousPairInfo // 已覆盖的并发对 (key -> pair)
	SusConPairs     map[string]*feedback.SuspiciousPairInfo // 可疑的并发对 (key -> pair)
	TryPairs        map[string]*feedback.SuspiciousPairInfo // 上次fuzzing输入的并发对 (key -> pair)
	FeedbackPair    map[string]*feedback.SuspiciousPairInfo // fuzzing结束反馈的并发对 (key -> pair)
	// 稳定性追踪：渐进冷却 + 定期重探测
	coolState           CoolDownState  // 当前冷却状态
	coolStateRounds     int            // 当前冷却状态持续轮次（pair 集合未变的轮次）
	coolSkipCount       int            // 冷却采样计数器
	roundsSinceLastFull int            // 距上次全量收集的轮次
	reProbeInterval     int            // 重探测间隔，默认 25
	prevPairCount       int            // 上一轮的总 pair 数
	execCount           uint32         // Get() 调用次数，用于交替反转，独立于反馈周期
	consecutiveTimeouts map[string]int // signalKey -> 连续超时次数
}

// NewCorpusPair 初始化CorpusPair
func NewCorpusPair() *CorpusPair {
	p := CorpusPair{}
	p.done = 0
	p.execCount = 0
	p.isReverse = true
	p.coolState = StateHot
	p.reProbeInterval = 25
	p.CoveredConPairs = make(map[string]*feedback.SuspiciousPairInfo)
	p.SusConPairs = make(map[string]*feedback.SuspiciousPairInfo)
	p.TryPairs = make(map[string]*feedback.SuspiciousPairInfo)
	p.FeedbackPair = make(map[string]*feedback.SuspiciousPairInfo)
	p.consecutiveTimeouts = make(map[string]int)
	return &p
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

	// 统一反转：每两次调用交替方向，不依赖反馈周期的稳定状态
	if p.isReverse {
		cnt := atomic.AddUint32(&p.execCount, 1)
		if cnt%2 == 0 {
			for i, pair := range result {
				result[i] = &feedback.SuspiciousPairInfo{
					FuncID1: pair.FuncID2, FuncID2: pair.FuncID1,
					CallLoc1: pair.CallLoc2, CallLoc2: pair.CallLoc1,
					Confidence: pair.Confidence,
					SourceType: pair.SourceType,
					IsObserved: pair.IsObserved,
				}
			}
		}
	}

	return &feedback.InputPair{
		TryPair:     result,
		RecordStack: p.shouldRecord(),
	}
}

// AddPair 添加并发对，按可信度分类并自动去重
// 1. 将 feedPair 分为 covered 和 suspicious
// 2. 找出 covered 和 TryPairs 的交集 a
// 3. 从 TryPairs 和 SusConPairs 中删除交集 a
// 4. 将 covered 加入 CoveredConPairs
func (p *CorpusPair) AddPair(feedPair []*feedback.SuspiciousPairInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()

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

	// 第七步：渐进冷却判定
	currentCount := len(p.SusConPairs) + len(p.CoveredConPairs)
	if currentCount != p.prevPairCount && currentCount > 0 {
		// 发现新 pair，立即回到 HOT
		if p.coolState != StateHot {
			fmt.Printf("[cooldown] new pairs discovered, reset: %v → HOT\n", p.coolState)
		}
		p.coolState = StateHot
		p.coolStateRounds = 0
		p.coolSkipCount = 0
		p.roundsSinceLastFull = 0
	} else if currentCount > 0 {
		// 集合未变，推进冷却
		p.coolStateRounds++
		// 每 3 轮不变则降一级
		if p.coolStateRounds >= 3 {
			switch p.coolState {
			case StateHot:
				p.coolState = StateWarm
				p.coolSkipCount = 0
				fmt.Println("[cooldown] HOT → WARM")
			case StateWarm:
				p.coolState = StateCool
				p.coolSkipCount = 0
				fmt.Println("[cooldown] WARM → COOL")
			case StateCool:
				p.coolState = StateCold
				p.coolSkipCount = 0
				fmt.Println("[cooldown] COOL → COLD")
			}
			p.coolStateRounds = 0
		}
	}
	p.prevPairCount = currentCount
	p.UpdateTryPairs()
}

// UpdateTryPairs 从 SusConPairs 中增量添加最多5个可疑并发对到 TryPairs
// 不会清空 TryPairs，只添加不存在的新项
// 注意：调用方（AddPair）已持有 p.mu 写锁，此处不再加锁
func (p *CorpusPair) UpdateTryPairs() {
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

// ApplySignals 应用调度有效性信号（从 stdout 解析的轻量反馈）
// 仅处理 func 级别信号；op 级别信号（COVERED_OP/TIMEOUT_OP）暂不处理
//
// {COVERED} → 该 pair 调度成功，重置超时计数
// {TIMEOUT} → 该 pair 调度失败，累计连续超时次数
//   连续超时 >= maxTimeouts 则从 TryPairs 移除（降级回 SusConPairs）
func (p *CorpusPair) ApplySignals(signals []*feedback.CoverageSignal) {
	const maxTimeouts = 10 // 连续超时阈值

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, sig := range signals {
		if sig == nil {
			continue
		}
		// 只处理 func 级别信号
		if sig.Kind != feedback.SignalFuncCovered && sig.Kind != feedback.SignalFuncTimeout {
			continue
		}

		key := signalKey(sig.PreID, sig.NextID)

		if sig.Success {
			// 调度成功：重置超时计数
			delete(p.consecutiveTimeouts, key)
		} else {
			// 调度超时：累计并检查阈值
			p.consecutiveTimeouts[key]++
			if p.consecutiveTimeouts[key] >= maxTimeouts {
				// 从 TryPairs 中移除匹配的 pair（按 FuncID 匹配）
				p.removeFromTryByFuncIDs(sig.PreID, sig.NextID)
				delete(p.consecutiveTimeouts, key)
				fmt.Printf("[signal] pair %v-%v removed from TryPairs after %d timeouts\n", sig.PreID, sig.NextID, maxTimeouts)
			}
		}
	}
}

// removeFromTryByFuncIDs 从 TryPairs 中移除匹配指定 FuncID 对的条目
func (p *CorpusPair) removeFromTryByFuncIDs(id1, id2 uint64) {
	for key, pair := range p.TryPairs {
		if (pair.FuncID1 == id1 && pair.FuncID2 == id2) ||
			(pair.FuncID1 == id2 && pair.FuncID2 == id1) {
			delete(p.TryPairs, key)
			return
		}
	}
}

// shouldRecord 根据冷却状态和重探测周期决定本轮是否记录调用栈
func (p *CorpusPair) shouldRecord() bool {
	p.coolSkipCount++
	p.roundsSinceLastFull++

	// 定期重探测：每 reProbeInterval 轮强制全量收集
	if p.roundsSinceLastFull >= p.reProbeInterval {
		p.roundsSinceLastFull = 0
		if p.coolState != StateHot {
			fmt.Printf("[re-probe] force full collection at state %v\n", p.coolState)
		}
		return true
	}

	// 渐进冷却采样
	switch p.coolState {
	case StateHot:
		return true
	case StateWarm:
		return p.coolSkipCount%2 == 0
	case StateCool:
		return p.coolSkipCount%4 == 0
	case StateCold:
		return p.coolSkipCount%8 == 0
	}
	return true
}
