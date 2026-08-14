package fuzzer

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"

	"toolkit/pkg/feedback"
)

// funcSignalKey 由两个函数 ID 生成信号匹配键。
func funcSignalKey(preID, nextID uint64) string {
	return pairSignalKey(preID, nextID)
}

// funcEdgeKey 是函数调用者-被调用者边的去重键。
type funcEdgeKey struct {
	caller uint64
	callee uint64
}

// CorpusFunc 管理函数级别的并发对发现与调度。
// 对标 CorpusGort，但使用函数 ID（SuspiciousPairInfo）
// 而非 goroutine ID（GortPairInfo）。
type CorpusFunc struct {
	mu              sync.RWMutex
	CoveredConPairs map[string]*feedback.SuspiciousPairInfo
	SusConPairs     map[string]*feedback.SuspiciousPairInfo
	InfeasiblePairs map[string]*feedback.SuspiciousPairInfo
	TryPairs        map[string]*feedback.SuspiciousPairInfo
	pairTimeouts    map[string]int
	selectNum       int

	callers  map[uint64]map[uint64]struct{} // callee -> set of callers
	callees  map[uint64]map[uint64]struct{} // caller -> set of callees
	edgeHits map[funcEdgeKey]uint64
	edgeRuns map[funcEdgeKey]uint64

	preExecRound  uint32
	prevPairTotal int
	stableCount   int

	phase *uint32
}

const (
	funcDefaultStableThreshold = 3
	funcMaxTimeouts            = 5
	funcMaxSelectNum           = 64
)

// NewCorpusFunc 创建函数级别的并发对语料库。
func NewCorpusFunc(phase *uint32) *CorpusFunc {
	return &CorpusFunc{
		CoveredConPairs: make(map[string]*feedback.SuspiciousPairInfo),
		SusConPairs:     make(map[string]*feedback.SuspiciousPairInfo),
		InfeasiblePairs: make(map[string]*feedback.SuspiciousPairInfo),
		TryPairs:        make(map[string]*feedback.SuspiciousPairInfo),
		pairTimeouts:    make(map[string]int),
		selectNum:       1,
		callers:         make(map[uint64]map[uint64]struct{}),
		callees:         make(map[uint64]map[uint64]struct{}),
		edgeHits:        make(map[funcEdgeKey]uint64),
		edgeRuns:        make(map[funcEdgeKey]uint64),
		phase:           phase,
	}
}

// Get 返回 TryPairs 作为下一次执行的输入。
func (p *CorpusFunc) Get() *feedback.InputPair {
	if atomic.LoadUint32(p.phase) == uint32(0) {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make([]*feedback.SuspiciousPairInfo, 0, len(p.TryPairs))
	for _, pair := range p.TryPairs {
		result = append(result, pair)
	}
	return &feedback.InputPair{TryPair: result}
}

// --- PairCorpus 接口方法 ---

// GetInput 实现了 PairCorpus 接口。
func (p *CorpusFunc) GetInput() *PairInput {
	fp := p.Get()
	if fp == nil || len(fp.TryPair) == 0 {
		return nil
	}
	return &PairInput{Pairs: funcToConcurrencyPairs(fp.TryPair)}
}

// AddConcurrencyPairs 实现了 PairCorpus 接口。
func (p *CorpusFunc) AddConcurrencyPairs(pairs []feedback.ConcurrencyPair) {
	p.AddPair(toFuncPairs(pairs))
}

// AddConcurrencyEdges 实现了 PairCorpus 接口。
func (p *CorpusFunc) AddConcurrencyEdges(edges []feedback.ConcurrencyEdge) int {
	funcEdges := make([]*feedback.FuncEdge, 0, len(edges))
	for _, e := range edges {
		if fe, ok := e.(*feedback.FuncEdge); ok {
			funcEdges = append(funcEdges, fe)
		}
	}
	return p.AddFuncEdges(funcEdges)
}

// OnCoveredByOp 实现了 PairCorpus 接口。
// 函数对被覆盖时，经 byFunc 索引（[FB] 日志 fids 字段）增量生成 OP 种子，
// 与 goroutine 模式的 OnGortCovered 语义对齐。
func (p *CorpusFunc) OnCoveredByOp(opCorpus *CorpusOp, pairs []feedback.ConcurrencyPair) {
	opCorpus.OnFuncCovered(toFuncPairs(pairs))
}

// SnapshotCovered 返回当前已覆盖函数对的切片副本（持读锁，供外部安全读取）。
func (p *CorpusFunc) SnapshotCovered() []*feedback.SuspiciousPairInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	pairs := make([]*feedback.SuspiciousPairInfo, 0, len(p.CoveredConPairs))
	for _, pair := range p.CoveredConPairs {
		if pair != nil {
			pairs = append(pairs, pair)
		}
	}
	return pairs
}

// OnPreExecEnd 实现了 PairCorpus 接口：函数模式从已覆盖函数对生成 OP 种子。
func (p *CorpusFunc) OnPreExecEnd(opCorpus *CorpusOp) {
	opCorpus.TryEndPreExecForFunc(p)
}

// InPreExec 实现了 PairCorpus 接口。
func (p *CorpusFunc) InPreExec() bool {
	return atomic.LoadUint32(p.phase) == 0
}

// ModeName 实现了 PairCorpus 接口。
func (p *CorpusFunc) ModeName() string {
	return "function"
}

// ApplyConcurrencySignals 实现了 PairCorpus 接口。
func (p *CorpusFunc) ApplyConcurrencySignals(signals []*feedback.CoverageSignal) []feedback.ConcurrencyPair {
	return funcToConcurrencyPairs(p.ApplySignals(signals))
}

// AddPair 合并单次执行中观测/推测出的函数对。
func (p *CorpusFunc) AddPair(feedPair []*feedback.SuspiciousPairInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()

	suspectsChanged := false
	var newlyObserved []*feedback.SuspiciousPairInfo
	for _, pair := range feedPair {
		if pair == nil || pair.FuncID1 == 0 || pair.FuncID2 == 0 || pair.FuncID1 == pair.FuncID2 {
			continue
		}
		key := pair.PairKey()
		signalKey := pair.SignalKey()
		if pair.IsObserved {
			if hasFuncSignalPair(p.CoveredConPairs, signalKey) {
				continue
			}
			removeFuncSignalPairs(p.SusConPairs, signalKey, p.pairTimeouts, p.TryPairs)
			removeFuncSignalPairs(p.InfeasiblePairs, signalKey, nil, nil)
			p.CoveredConPairs[key] = pair
			newlyObserved = append(newlyObserved, pair)
		} else {
			if hasFuncSignalPair(p.CoveredConPairs, signalKey) ||
				hasFuncSignalPair(p.InfeasiblePairs, signalKey) {
				continue
			}
			found := false
			for _, existing := range p.SusConPairs {
				if existing == nil || existing.SignalKey() != signalKey {
					continue
				}
				found = true
				if pair.Confidence > existing.Confidence {
					existing.Confidence = pair.Confidence
					existing.SourceType = pair.SourceType
					suspectsChanged = true
				}
			}
			if !found {
				p.SusConPairs[key] = pair
				suspectsChanged = true
			}
		}
	}

	if atomic.LoadUint32(p.phase) == 1 {
		for _, pair := range newlyObserved {
			if p.inferFromAnchorLocked(pair) > 0 {
				suspectsChanged = true
			}
		}
	}
	if (suspectsChanged || len(newlyObserved) > 0) && atomic.LoadUint32(p.phase) == 1 {
		p.RefillTryPairs()
	}
}

// AddFuncEdges 合并单次执行中的函数调用者-被调用者边。
func (p *CorpusFunc) AddFuncEdges(edges []*feedback.FuncEdge) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	runHits := make(map[funcEdgeKey]uint64)
	for _, edge := range edges {
		if edge == nil || edge.Count == 0 || edge.Callee == 0 || edge.Caller == edge.Callee {
			continue
		}
		key := funcEdgeKey{caller: edge.Caller, callee: edge.Callee}
		runHits[key] += edge.Count
	}

	newEdges := 0
	for key, hits := range runHits {
		if _, exists := p.edgeHits[key]; !exists {
			newEdges++
			if p.callers[key.callee] == nil {
				p.callers[key.callee] = make(map[uint64]struct{})
			}
			p.callers[key.callee][key.caller] = struct{}{}
			if p.callees[key.caller] == nil {
				p.callees[key.caller] = make(map[uint64]struct{})
			}
			p.callees[key.caller][key.callee] = struct{}{}
		}
		p.edgeHits[key] += hits
		p.edgeRuns[key]++
	}

	if newEdges > 0 && atomic.LoadUint32(p.phase) == 1 {
		p.inferFromAllCoveredLocked()
		p.RefillTryPairs()
	}
	return newEdges
}

// TryEndPreExec 判断预执行种子收集是否应结束。
func (p *CorpusFunc) TryEndPreExec(maxRounds int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if atomic.LoadUint32(p.phase) == 1 {
		return
	}

	round := atomic.AddUint32(&p.preExecRound, 1)
	total := len(p.CoveredConPairs) + len(p.SusConPairs) + len(p.edgeHits)

	if total != p.prevPairTotal {
		p.prevPairTotal = total
		p.stableCount = 0
	} else {
		p.stableCount++
	}

	if p.stableCount >= funcDefaultStableThreshold || int(round) >= maxRounds {
		atomic.StoreUint32(p.phase, 1)
		p.RefillTryPairs()
		fmt.Printf("[PRESTAGE] Pre-execution finished (func mode), total pairs: cover=%d, suspect=%d, edges=%d\n",
			len(p.CoveredConPairs), len(p.SusConPairs), len(p.edgeHits))
	}
}

// RefillTryPairs 用得分最高的 SusConPairs 替换 TryPairs。
func (p *CorpusFunc) RefillTryPairs() {
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

	sort.Slice(candidates, func(i, j int) bool {
		iScore := p.funcScore(candidates[i].key)
		jScore := p.funcScore(candidates[j].key)
		if iScore == jScore {
			return candidates[i].key < candidates[j].key
		}
		return iScore > jScore
	})

	n := p.selectNum
	if n > len(candidates) {
		n = len(candidates)
	}
	for i := 0; i < n; i++ {
		p.TryPairs[candidates[i].key] = candidates[i].pair
	}
}

func (p *CorpusFunc) funcScore(key string) float64 {
	pair, ok := p.SusConPairs[key]
	if !ok {
		return -1
	}
	return pair.Confidence*10 - float64(p.pairTimeouts[key])*2
}

func hasFuncSignalPair(pairs map[string]*feedback.SuspiciousPairInfo, signalKey string) bool {
	for _, pair := range pairs {
		if pair != nil && pair.SignalKey() == signalKey {
			return true
		}
	}
	return false
}

func removeFuncSignalPairs(
	pairs map[string]*feedback.SuspiciousPairInfo,
	signalKey string,
	timeouts map[string]int,
	tryPairs map[string]*feedback.SuspiciousPairInfo,
) {
	for key, pair := range pairs {
		if pair == nil || pair.SignalKey() != signalKey {
			continue
		}
		delete(pairs, key)
		if timeouts != nil {
			delete(timeouts, key)
		}
		if tryPairs != nil {
			delete(tryPairs, key)
		}
	}
}

func (p *CorpusFunc) addInferredPairLocked(fid1, fid2 uint64, confidence float64, sourceType string) bool {
	if fid1 == 0 || fid2 == 0 || fid1 == fid2 {
		return false
	}

	signalKey := funcSignalKey(fid1, fid2)
	if hasFuncSignalPair(p.CoveredConPairs, signalKey) || hasFuncSignalPair(p.InfeasiblePairs, signalKey) {
		return false
	}

	found := false
	changed := false
	for _, pair := range p.SusConPairs {
		if pair == nil || pair.SignalKey() != signalKey {
			continue
		}
		found = true
		if pair.Confidence < confidence {
			pair.Confidence = confidence
			pair.SourceType = sourceType
			changed = true
		}
	}
	if found {
		return changed
	}

	if fid2 < fid1 {
		fid1, fid2 = fid2, fid1
	}
	pair := &feedback.SuspiciousPairInfo{
		FuncID1:    fid1,
		FuncID2:    fid2,
		Confidence: confidence,
		SourceType: sourceType,
		IsObserved: false,
	}
	p.SusConPairs[pair.PairKey()] = pair
	return true
}

func (p *CorpusFunc) inferFromAnchorLocked(anchor *feedback.SuspiciousPairInfo) int {
	if anchor == nil || anchor.FuncID1 == 0 || anchor.FuncID2 == 0 || anchor.FuncID1 == anchor.FuncID2 {
		return 0
	}

	changed := 0
	inferSide := func(fid, other uint64) {
		for caller := range p.callers[fid] {
			if p.addInferredPairLocked(caller, other, 0.5, "fuzz_inferred_adjacent") {
				changed++
			}
			for sibling := range p.callees[caller] {
				if sibling == fid {
					continue
				}
				if p.addInferredPairLocked(sibling, other, 0.5, "fuzz_inferred_sibling") {
					changed++
				}
			}
		}
		for callee := range p.callees[fid] {
			if p.addInferredPairLocked(callee, other, 0.3, "fuzz_inferred_adjacent") {
				changed++
			}
		}
	}

	inferSide(anchor.FuncID1, anchor.FuncID2)
	inferSide(anchor.FuncID2, anchor.FuncID1)
	return changed
}

func (p *CorpusFunc) inferFromAllCoveredLocked() int {
	changed := 0
	for _, pair := range p.CoveredConPairs {
		changed += p.inferFromAnchorLocked(pair)
	}
	return changed
}

// ApplySignals 应用函数级别的覆盖/超时信号。
func (p *CorpusFunc) ApplySignals(signals []*feedback.CoverageSignal) []*feedback.SuspiciousPairInfo {
	p.mu.Lock()
	defer p.mu.Unlock()

	coveredSigKeys := make(map[string]struct{})
	timeoutSigKeys := make(map[string]struct{})
	for _, sig := range signals {
		if sig == nil {
			continue
		}
		if sig.Kind != feedback.SignalPairCovered && sig.Kind != feedback.SignalPairTimeout {
			continue
		}
		sk := funcSignalKey(sig.PreID, sig.NextID)
		if sig.Success {
			coveredSigKeys[sk] = struct{}{}
		} else {
			timeoutSigKeys[sk] = struct{}{}
		}
	}

	for sk := range timeoutSigKeys {
		for tryKey, tryPair := range p.TryPairs {
			if tryPair.SignalKey() != sk {
				continue
			}
			p.pairTimeouts[tryKey]++
			if p.pairTimeouts[tryKey] >= funcMaxTimeouts {
				p.InfeasiblePairs[tryKey] = tryPair
				delete(p.SusConPairs, tryKey)
				delete(p.pairTimeouts, tryKey)
				fmt.Printf("[func_signal] pair %v-%v infeasible after %d timeouts\n",
					tryPair.FuncID1, tryPair.FuncID2, funcMaxTimeouts)
			}
		}
	}

	intersection := make(map[string]*feedback.SuspiciousPairInfo)
	for tryKey, tryPair := range p.TryPairs {
		if _, ok := coveredSigKeys[tryPair.SignalKey()]; ok {
			intersection[tryKey] = tryPair
		}
	}

	if len(intersection) == 0 && len(timeoutSigKeys) > 0 {
		p.selectNum *= 2
		if p.selectNum > funcMaxSelectNum {
			p.selectNum = funcMaxSelectNum
		}
	}

	var newlyCovered []*feedback.SuspiciousPairInfo
	for key, pair := range intersection {
		pair.Confidence = 1.0
		pair.IsObserved = true
		pair.SourceType = "fuzz_verified"
		signalKey := pair.SignalKey()
		removeFuncSignalPairs(p.SusConPairs, signalKey, p.pairTimeouts, p.TryPairs)
		removeFuncSignalPairs(p.InfeasiblePairs, signalKey, nil, nil)
		p.CoveredConPairs[key] = pair
		newlyCovered = append(newlyCovered, pair)
	}
	for _, pair := range newlyCovered {
		p.inferFromAnchorLocked(pair)
	}

	p.RefillTryPairs()
	return newlyCovered
}
