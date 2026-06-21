package fuzzer

import (
	"fmt"
	"sort"
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

type CorpusPair struct {
	mu              sync.RWMutex
	isReverse       bool                                    // 是否反转pair
	done            uint32                                  // 0=预执行中, 1=预执行完成
	CoveredConPairs map[string]*feedback.SuspiciousPairInfo // 已覆盖的并发对 (key -> pair)
	SusConPairs     map[string]*feedback.SuspiciousPairInfo // 可疑的并发对 (key -> pair)
	InfeasiblePairs map[string]*feedback.SuspiciousPairInfo // 无法覆盖的并发对 (key -> pair)
	TryPairs        map[string]*feedback.SuspiciousPairInfo // 本轮fuzzing输入的并发对 (key -> pair)，全量替换
	//FeedbackPair    map[string]*feedback.SuspiciousPairInfo // fuzzing结束反馈的并发对 (key -> pair)
	execCount    uint32         // Get() 调用次数，用于交替反转
	pairTimeouts map[string]int // pairKey -> 累计超时次数
	selectNum    int            // 从 SusConPairs 选取的数量，初始=1，自适应调整（上限64）

	preExecRound  uint32 // 预执行当前轮次
	prevPairTotal int    // 上一轮 pair 总数
	stableCount   int    // 连续不变轮数
}

// NewCorpusPair 初始化CorpusPair
func NewCorpusPair() *CorpusPair {
	p := CorpusPair{}
	p.done = 0
	p.execCount = 0
	p.isReverse = true
	p.CoveredConPairs = make(map[string]*feedback.SuspiciousPairInfo)
	p.SusConPairs = make(map[string]*feedback.SuspiciousPairInfo)
	p.InfeasiblePairs = make(map[string]*feedback.SuspiciousPairInfo)
	p.TryPairs = make(map[string]*feedback.SuspiciousPairInfo)
	p.pairTimeouts = make(map[string]int)
	p.selectNum = 1
	p.preExecRound = 0
	p.prevPairTotal = 0
	p.stableCount = 0
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

// AddPair 添加并发对，仅用于填充 CoveredConPairs 和 SusConPairs
// 预执行阶段使用，不涉及 TryPairs 交互
func (p *CorpusPair) AddPair(feedPair []*feedback.SuspiciousPairInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, pair := range feedPair {
		if pair == nil {
			continue
		}
		key := pairKey(pair)
		if pair.IsObserved {
			// observed → 直接覆盖写入 CoveredConPairs（最可靠）
			p.CoveredConPairs[key] = pair
		} else {
			// inferred → 仅首次写入 SusConPairs（保留最早的 CallLoc）
			if _, ok := p.SusConPairs[key]; !ok {
				p.SusConPairs[key] = pair
			}
		}
	}
}

const defaultStableThreshold = 3

// TryEndPreExec 判断预执行是否结束，结束后设置 done=1 并首次填充 TryPairs
func (p *CorpusPair) TryEndPreExec(maxRounds int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if atomic.LoadUint32(&p.done) == 1 {
		return true // 已结束
	}

	round := atomic.AddUint32(&p.preExecRound, 1)
	total := len(p.CoveredConPairs) + len(p.SusConPairs)

	if total != p.prevPairTotal {
		p.prevPairTotal = total
		p.stableCount = 0
	} else {
		p.stableCount++
	}

	if p.stableCount >= defaultStableThreshold || int(round) >= maxRounds {
		atomic.StoreUint32(&p.done, 1)
		p.RefillTryPairs() // 首次从 SusConPairs 填充 TryPairs
		return true
	}
	return false
}

const (
	maxTimeouts  = 5  // 超时上限，达到后移入 InfeasiblePairs
	maxSelectNum = 64 // selectNum 翻倍上限
)

// RefillTryPairs 全量替换 TryPairs，从 SusConPairs 按优先级选取 selectNum 个
// 调用方（TryEndPreExec / ApplySignals）已持有 p.mu 写锁，此处不再加锁
// todo 可能会出现种子饿死的问题
func (p *CorpusPair) RefillTryPairs() {
	p.TryPairs = make(map[string]*feedback.SuspiciousPairInfo)

	if len(p.SusConPairs) == 0 {
		return
	}

	type cand struct {
		key  string
		pair *feedback.SuspiciousPairInfo
	}
	candidates := make([]cand, 0, len(p.SusConPairs))
	for key, pair := range p.SusConPairs {
		candidates = append(candidates, cand{key, pair})
	}

	// 按 score 降序：confidence 高 + timeout 少 + observed/parent 加成
	sort.Slice(candidates, func(i, j int) bool {
		return p.score(candidates[i].key) > p.score(candidates[j].key)
	})

	n := p.selectNum
	if n > len(candidates) {
		n = len(candidates)
	}
	for i := 0; i < n; i++ {
		p.TryPairs[candidates[i].key] = candidates[i].pair
	}
}

// score 计算 pair 的综合调度优先级（key 为 pairKey）
// 维度：Confidence × 10 − Timeout × 2
func (p *CorpusPair) score(key string) float64 {
	pair, ok := p.SusConPairs[key]
	if !ok {
		return -1
	}
	return pair.Confidence*10 - float64(p.pairTimeouts[key])*2
}

// ApplySignals 应用调度有效性信号
//
// 流程：
//  1. 对 TIMEOUT 信号匹配 TryPairs 中的条目，累计超时（≥maxTimeouts → InfeasiblePairs）
//  2. 计算 COVERED ∩ TryPairs 交集
//  3. 交集为空 → selectNum 翻倍（上限 maxSelectNum）
//  4. 交集非空 → 从 SusConPairs 删除交集，加入 CoveredConPairs
//  5. 全量替换 TryPairs
func (p *CorpusPair) ApplySignals(signals []*feedback.CoverageSignal) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// ① 信号分类
	coveredSigKeys := make(map[string]struct{})
	timeoutSigKeys := make(map[string]struct{})
	for _, sig := range signals {
		if sig == nil {
			continue
		}
		if sig.Kind != feedback.SignalFuncCovered && sig.Kind != feedback.SignalFuncTimeout {
			continue
		}
		sk := signalKey(sig.PreID, sig.NextID)
		if sig.Success {
			coveredSigKeys[sk] = struct{}{}
		} else {
			timeoutSigKeys[sk] = struct{}{}
		}
	}

	// ② 超时计数：通过 signalKey 匹配 TryPairs 中的 pairKey
	for sk := range timeoutSigKeys {
		for tryKey, tryPair := range p.TryPairs {
			if signalKey(tryPair.FuncID1, tryPair.FuncID2) != sk {
				continue
			}
			p.pairTimeouts[tryKey]++
			if p.pairTimeouts[tryKey] >= maxTimeouts {
				// 移入 InfeasiblePairs，从 SusConPairs 删除
				p.InfeasiblePairs[tryKey] = tryPair
				delete(p.SusConPairs, tryKey)
				delete(p.pairTimeouts, tryKey)
				fmt.Printf("[signal] pair %v-%v|%s:%d-%s:%d infeasible after %d timeouts\n",
					tryPair.FuncID1, tryPair.FuncID2,
					tryPair.CallLoc1.File, tryPair.CallLoc1.Line,
					tryPair.CallLoc2.File, tryPair.CallLoc2.Line,
					maxTimeouts)
			}
		}
	}

	// ③ 计算 COVERED ∩ TryPairs
	intersection := make(map[string]*feedback.SuspiciousPairInfo)
	for tryKey, tryPair := range p.TryPairs {
		if _, ok := coveredSigKeys[signalKey(tryPair.FuncID1, tryPair.FuncID2)]; ok {
			intersection[tryKey] = tryPair
		}
	}

	// ④ 根据交集处理
	if len(intersection) == 0 && len(timeoutSigKeys) > 0 {
		// 无覆盖且本轮有超时 → selectNum 翻倍
		p.selectNum *= 2
		if p.selectNum > maxSelectNum {
			p.selectNum = maxSelectNum
		}
	}
	// 有覆盖 → selectNum 不变，从 SusConPairs 删除交集，加入 CoveredConPairs
	for key, pair := range intersection {
		p.CoveredConPairs[key] = pair
		delete(p.SusConPairs, key)
		delete(p.pairTimeouts, key)
	}

	// ⑤ 全量替换 TryPairs
	p.RefillTryPairs()
}

// shouldRecord 决定本轮是否记录调用栈（始终返回 true，全量收集）
func (p *CorpusPair) shouldRecord() bool {
	return true
}
