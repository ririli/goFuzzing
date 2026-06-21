package fuzzer

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"

	"toolkit/pkg/feedback"
)

// gortPairKey 生成goroutine并发对的唯一键（基于 Gid 和 CallLoc）
func gortPairKey(pair *feedback.GortPairInfo) string {
	var gid1, gid2 uint64
	var loc1, loc2 feedback.CallLocationInfo

	if pair.Gid1 <= pair.Gid2 {
		gid1, gid2 = pair.Gid1, pair.Gid2
		loc1, loc2 = pair.CallLoc1, pair.CallLoc2
	} else {
		gid1, gid2 = pair.Gid2, pair.Gid1
		loc1, loc2 = pair.CallLoc2, pair.CallLoc1
	}

	return fmt.Sprintf("%d-%d|%s:%d-%s:%d",
		gid1, gid2,
		loc1.File, loc1.Line,
		loc2.File, loc2.Line)
}

// gortSignalKey 生成goroutine调度信号的查找键（仅基于 Gid，不含 CallLoc）
func gortSignalKey(preID, nextID uint64) string {
	if preID <= nextID {
		return fmt.Sprintf("%d-%d", preID, nextID)
	}
	return fmt.Sprintf("%d-%d", nextID, preID)
}

// CorpusGort goroutine级别并发对语料库
// 对标 CorpusPair，但使用 GortPairInfo 替代 SuspiciousPairInfo
type CorpusGort struct {
	mu              sync.RWMutex
	isReverse       bool                              // 是否反转pair
	done            uint32                            // 0=预执行中, 1=预执行完成
	CoveredConPairs map[string]*feedback.GortPairInfo // 已覆盖的goroutine并发对 (Rule 0直接观测)
	SusConPairs     map[string]*feedback.GortPairInfo // 可疑的goroutine并发对 (Rule 1-4推测)
	InfeasiblePairs map[string]*feedback.GortPairInfo // 无法覆盖的goroutine并发对 (超时过多或已完成)
	TryPairs        map[string]*feedback.GortPairInfo // 本轮fuzzing输入的goroutine并发对
	execCount       uint32                            // Get() 调用次数，用于交替反转
	pairTimeouts    map[string]int                    // pairKey -> 累计超时次数
	selectNum       int                               // 从 SusConPairs 选取的数量，初始=1，自适应调整（上限64）

	preExecRound  uint32 // 预执行当前轮次
	prevPairTotal int    // 上一轮 pair 总数
	stableCount   int    // 连续不变轮数
}

// NewCorpusGort 初始化CorpusGort
func NewCorpusGort() *CorpusGort {
	p := CorpusGort{}
	p.done = 0
	p.execCount = 0
	p.isReverse = true
	p.CoveredConPairs = make(map[string]*feedback.GortPairInfo)
	p.SusConPairs = make(map[string]*feedback.GortPairInfo)
	p.InfeasiblePairs = make(map[string]*feedback.GortPairInfo)
	p.TryPairs = make(map[string]*feedback.GortPairInfo)
	p.pairTimeouts = make(map[string]int)
	p.selectNum = 1
	p.preExecRound = 0
	p.prevPairTotal = 0
	p.stableCount = 0
	return &p
}

// Get 获取 TryPairs 作为 InputGortPair 返回
func (p *CorpusGort) Get() *feedback.InputGortPair {
	if atomic.LoadUint32(&p.done) == uint32(0) {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make([]*feedback.GortPairInfo, 0, len(p.TryPairs))
	for _, pair := range p.TryPairs {
		result = append(result, pair)
	}

	// 统一反转：每两次调用交替方向
	if p.isReverse {
		cnt := atomic.AddUint32(&p.execCount, 1)
		if cnt%2 == 0 {
			for i, pair := range result {
				result[i] = &feedback.GortPairInfo{
					Gid1: pair.Gid2, Gid2: pair.Gid1,
					CallLoc1: pair.CallLoc2, CallLoc2: pair.CallLoc1,
					Confidence: pair.Confidence,
					SourceType: pair.SourceType,
					IsObserved: pair.IsObserved,
				}
			}
		}
	}

	return &feedback.InputGortPair{
		TryPair:     result,
		RecordStack: p.shouldRecord(),
	}
}

// AddPair 添加goroutine并发对到对应集合
// IsObserved → CoveredConPairs（Rule 0直接观测）
// !IsObserved → SusConPairs（Rule 1-4推测）
func (p *CorpusGort) AddPair(feedPair []*feedback.GortPairInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, pair := range feedPair {
		if pair == nil {
			continue
		}
		key := gortPairKey(pair)
		if pair.IsObserved {
			p.CoveredConPairs[key] = pair
		} else {
			if _, ok := p.SusConPairs[key]; !ok {
				p.SusConPairs[key] = pair
			}
		}
	}
}

const gortDefaultStableThreshold = 3

// TryEndPreExec 判断预执行是否结束
func (p *CorpusGort) TryEndPreExec(maxRounds int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if atomic.LoadUint32(&p.done) == 1 {
		return true
	}

	round := atomic.AddUint32(&p.preExecRound, 1)
	total := len(p.CoveredConPairs) + len(p.SusConPairs)

	if total != p.prevPairTotal {
		p.prevPairTotal = total
		p.stableCount = 0
	} else {
		p.stableCount++
	}

	if p.stableCount >= gortDefaultStableThreshold || int(round) >= maxRounds {
		atomic.StoreUint32(&p.done, 1)
		p.RefillTryPairs()
		return true
	}
	return false
}

const (
	gortMaxTimeouts  = 5
	gortMaxSelectNum = 64
)

// RefillTryPairs 全量替换 TryPairs，从 SusConPairs 按优先级选取 selectNum 个
func (p *CorpusGort) RefillTryPairs() {
	p.TryPairs = make(map[string]*feedback.GortPairInfo)

	if len(p.SusConPairs) == 0 {
		return
	}

	type cand struct {
		key  string
		pair *feedback.GortPairInfo
	}
	candidates := make([]cand, 0, len(p.SusConPairs))
	for key, pair := range p.SusConPairs {
		candidates = append(candidates, cand{key, pair})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return p.gortScore(candidates[i].key) > p.gortScore(candidates[j].key)
	})

	n := p.selectNum
	if n > len(candidates) {
		n = len(candidates)
	}
	for i := 0; i < n; i++ {
		p.TryPairs[candidates[i].key] = candidates[i].pair
	}
}

// gortScore 计算goroutine pair的综合调度优先级
func (p *CorpusGort) gortScore(key string) float64 {
	pair, ok := p.SusConPairs[key]
	if !ok {
		return -1
	}
	return pair.Confidence*10 - float64(p.pairTimeouts[key])*2
}

// ApplySignals 应用goroutine调度有效性信号
func (p *CorpusGort) ApplySignals(signals []*feedback.CoverageSignal) {
	p.mu.Lock()
	defer p.mu.Unlock()

	coveredSigKeys := make(map[string]struct{})
	timeoutSigKeys := make(map[string]struct{})
	for _, sig := range signals {
		if sig == nil {
			continue
		}
		if sig.Kind != feedback.SignalFuncCovered && sig.Kind != feedback.SignalFuncTimeout {
			continue
		}
		sk := gortSignalKey(sig.PreID, sig.NextID)
		if sig.Success {
			coveredSigKeys[sk] = struct{}{}
		} else {
			timeoutSigKeys[sk] = struct{}{}
		}
	}

	// 超时计数
	for sk := range timeoutSigKeys {
		for tryKey, tryPair := range p.TryPairs {
			if gortSignalKey(tryPair.Gid1, tryPair.Gid2) != sk {
				continue
			}
			p.pairTimeouts[tryKey]++
			if p.pairTimeouts[tryKey] >= gortMaxTimeouts {
				p.InfeasiblePairs[tryKey] = tryPair
				delete(p.SusConPairs, tryKey)
				delete(p.pairTimeouts, tryKey)
				fmt.Printf("[gort_signal] pair %v-%v|%s:%d-%s:%d infeasible after %d timeouts\n",
					tryPair.Gid1, tryPair.Gid2,
					tryPair.CallLoc1.File, tryPair.CallLoc1.Line,
					tryPair.CallLoc2.File, tryPair.CallLoc2.Line,
					gortMaxTimeouts)
			}
		}
	}

	// 计算 COVERED ∩ TryPairs
	intersection := make(map[string]*feedback.GortPairInfo)
	for tryKey, tryPair := range p.TryPairs {
		if _, ok := coveredSigKeys[gortSignalKey(tryPair.Gid1, tryPair.Gid2)]; ok {
			intersection[tryKey] = tryPair
		}
	}

	if len(intersection) == 0 && len(timeoutSigKeys) > 0 {
		p.selectNum *= 2
		if p.selectNum > gortMaxSelectNum {
			p.selectNum = gortMaxSelectNum
		}
	}
	for key, pair := range intersection {
		p.CoveredConPairs[key] = pair
		delete(p.SusConPairs, key)
		delete(p.pairTimeouts, key)
	}

	p.RefillTryPairs()
}

func (p *CorpusGort) shouldRecord() bool {
	return true
}
