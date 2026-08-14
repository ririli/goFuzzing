package fuzzer

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"

	"toolkit/pkg/feedback"
)

// gortSignalKey 生成goroutine调度信号的查找键（仅基于 Gid，不含 CallLoc）
func gortSignalKey(preID, nextID uint64) string {
	return pairSignalKey(preID, nextID)
}

type gortEdgeKey struct {
	parent uint64
	child  uint64
}

// CorpusGort goroutine级别并发对语料库
// 对标 CorpusPair，但使用 GortPairInfo 替代 SuspiciousPairInfo
type CorpusGort struct {
	mu              sync.RWMutex
	CoveredConPairs map[string]*feedback.GortPairInfo // 已覆盖的goroutine并发对 (Rule 0直接观测)
	SusConPairs     map[string]*feedback.GortPairInfo // 可疑的goroutine并发对 (Rule 1-4推测)
	InfeasiblePairs map[string]*feedback.GortPairInfo // 无法覆盖的goroutine并发对 (超时过多或已完成)
	TryPairs        map[string]*feedback.GortPairInfo // 本轮fuzzing输入的goroutine并发对
	pairTimeouts    map[string]int                    // pairKey -> 累计超时次数
	selectNum       int                               // 从 SusConPairs 选取的数量，初始=1，自适应调整（上限64）
	parents         map[uint64]map[uint64]struct{}    // child gid -> observed parent gids
	children        map[uint64]map[uint64]struct{}    // parent gid -> observed child gids
	edgeHits        map[gortEdgeKey]uint64            // edge -> dynamic occurrence count
	edgeRuns        map[gortEdgeKey]uint64            // edge -> number of executions that observed it

	preExecRound  uint32 // 预执行当前轮次
	prevPairTotal int    // 上一轮 pair + topology edge 总数
	stableCount   int    // 连续不变轮数

	phase *uint32 // 指向 cfg.Phase，预执行→fuzzing 转换时写入
}

// NewCorpusGort 初始化CorpusGort
func NewCorpusGort(phase *uint32) *CorpusGort {
	p := CorpusGort{}
	p.CoveredConPairs = make(map[string]*feedback.GortPairInfo)
	p.SusConPairs = make(map[string]*feedback.GortPairInfo)
	p.InfeasiblePairs = make(map[string]*feedback.GortPairInfo)
	p.TryPairs = make(map[string]*feedback.GortPairInfo)
	p.pairTimeouts = make(map[string]int)
	p.parents = make(map[uint64]map[uint64]struct{})
	p.children = make(map[uint64]map[uint64]struct{})
	p.edgeHits = make(map[gortEdgeKey]uint64)
	p.edgeRuns = make(map[gortEdgeKey]uint64)
	p.selectNum = 1
	p.preExecRound = 0
	p.prevPairTotal = 0
	p.stableCount = 0
	p.phase = phase
	return &p
}

// Get 获取 TryPairs 作为 InputGortPair 返回
func (p *CorpusGort) Get() *feedback.InputGortPair {
	if atomic.LoadUint32(p.phase) == uint32(0) {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make([]*feedback.GortPairInfo, 0, len(p.TryPairs))
	for _, pair := range p.TryPairs {
		result = append(result, pair)
	}

	return &feedback.InputGortPair{
		TryPair: result,
	}
}

// --- PairCorpus interface methods ---

// GetInput implements PairCorpus.
func (p *CorpusGort) GetInput() *PairInput {
	gp := p.Get()
	if gp == nil || len(gp.TryPair) == 0 {
		return nil
	}
	return &PairInput{Pairs: gortToConcurrencyPairs(gp.TryPair)}
}

// AddConcurrencyPairs implements PairCorpus. Converts and delegates to AddPair.
func (p *CorpusGort) AddConcurrencyPairs(pairs []feedback.ConcurrencyPair) {
	p.AddPair(toGortPairs(pairs))
}

// AddConcurrencyEdges implements PairCorpus. Converts and delegates to AddGortEdges.
func (p *CorpusGort) AddConcurrencyEdges(edges []feedback.ConcurrencyEdge) int {
	gortEdges := make([]*feedback.GortEdge, 0, len(edges))
	for _, e := range edges {
		if ge, ok := e.(*feedback.GortEdge); ok {
			gortEdges = append(gortEdges, ge)
		}
	}
	return p.AddGortEdges(gortEdges)
}

// OnCoveredByOp implements PairCorpus.
func (p *CorpusGort) OnCoveredByOp(opCorpus *CorpusOp, pairs []feedback.ConcurrencyPair) {
	opCorpus.OnGortCovered(toGortPairs(pairs))
}

// OnPreExecEnd implements PairCorpus: goroutine 模式从已覆盖对生成 OP 种子。
func (p *CorpusGort) OnPreExecEnd(opCorpus *CorpusOp) {
	opCorpus.TryEndPreExec(p)
}

// InPreExec implements PairCorpus.
func (p *CorpusGort) InPreExec() bool {
	return atomic.LoadUint32(p.phase) == 0
}

// ModeName implements PairCorpus.
func (p *CorpusGort) ModeName() string {
	return "goroutine"
}

// ApplyConcurrencySignals implements PairCorpus. Wraps ApplySignals.
func (p *CorpusGort) ApplyConcurrencySignals(signals []*feedback.CoverageSignal) []feedback.ConcurrencyPair {
	return gortToConcurrencyPairs(p.ApplySignals(signals))
}

// AddPair 添加goroutine并发对到对应集合
// IsObserved → CoveredConPairs（Rule 0直接观测）
// !IsObserved → SusConPairs（Rule 1-4推测）
func (p *CorpusGort) AddPair(feedPair []*feedback.GortPairInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()

	suspectsChanged := false
	var newlyObserved []*feedback.GortPairInfo
	for _, pair := range feedPair {
		if pair == nil || pair.Gid1 == 0 || pair.Gid2 == 0 || pair.Gid1 == pair.Gid2 {
			continue
		}
		key := feedback.GortPairKey(pair)
		signalKey := gortSignalKey(pair.Gid1, pair.Gid2)
		if pair.IsObserved {
			if hasGortSignalPair(p.CoveredConPairs, signalKey) {
				continue
			}
			removeGortSignalPairs(p.SusConPairs, signalKey, p.pairTimeouts, p.TryPairs)
			removeGortSignalPairs(p.InfeasiblePairs, signalKey, nil, nil)
			p.CoveredConPairs[key] = pair
			newlyObserved = append(newlyObserved, pair)
		} else {
			if hasGortSignalPair(p.CoveredConPairs, signalKey) ||
				hasGortSignalPair(p.InfeasiblePairs, signalKey) {
				continue
			}
			found := false
			for _, existing := range p.SusConPairs {
				if existing == nil || gortSignalKey(existing.Gid1, existing.Gid2) != signalKey {
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

// SnapshotCovered 返回当前已覆盖 goroutine 对的切片副本（持读锁，供外部安全读取）。
func (p *CorpusGort) SnapshotCovered() []*feedback.GortPairInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	pairs := make([]*feedback.GortPairInfo, 0, len(p.CoveredConPairs))
	for _, pair := range p.CoveredConPairs {
		if pair != nil {
			pairs = append(pairs, pair)
		}
	}
	return pairs
}

// AddGortEdges merges one execution's static goroutine topology into the corpus.
// The topology is monotonic across executions: missing edges never remove old data.
func (p *CorpusGort) AddGortEdges(edges []*feedback.GortEdge) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	runHits := make(map[gortEdgeKey]uint64)
	for _, edge := range edges {
		if edge == nil || edge.Count == 0 || edge.ChildGid == 0 || edge.ParentGid == edge.ChildGid {
			continue
		}
		key := gortEdgeKey{parent: edge.ParentGid, child: edge.ChildGid}
		runHits[key] += edge.Count
	}

	newEdges := 0
	for key, hits := range runHits {
		if _, exists := p.edgeHits[key]; !exists {
			newEdges++
			if p.parents[key.child] == nil {
				p.parents[key.child] = make(map[uint64]struct{})
			}
			p.parents[key.child][key.parent] = struct{}{}
			if p.children[key.parent] == nil {
				p.children[key.parent] = make(map[uint64]struct{})
			}
			p.children[key.parent][key.child] = struct{}{}
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

const gortDefaultStableThreshold = 3

// TryEndPreExec 判断预执行是否结束
func (p *CorpusGort) TryEndPreExec(maxRounds int) {
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

	if p.stableCount >= gortDefaultStableThreshold || int(round) >= maxRounds {
		atomic.StoreUint32(p.phase, 1)
		p.RefillTryPairs()
		fmt.Printf("[PRESTAGE] Pre-execution finished, total gort pairs: cover=%d, suspect=%d, topology_edges=%d\n",
			len(p.CoveredConPairs), len(p.SusConPairs), len(p.edgeHits))
	}
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
		iScore := p.gortScore(candidates[i].key)
		jScore := p.gortScore(candidates[j].key)
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

// gortScore 计算goroutine pair的综合调度优先级
func (p *CorpusGort) gortScore(key string) float64 {
	pair, ok := p.SusConPairs[key]
	if !ok {
		return -1
	}
	return pair.Confidence*10 - float64(p.pairTimeouts[key])*2
}

func hasGortSignalPair(pairs map[string]*feedback.GortPairInfo, signalKey string) bool {
	for _, pair := range pairs {
		if pair != nil && gortSignalKey(pair.Gid1, pair.Gid2) == signalKey {
			return true
		}
	}
	return false
}

func removeGortSignalPairs(
	pairs map[string]*feedback.GortPairInfo,
	signalKey string,
	timeouts map[string]int,
	tryPairs map[string]*feedback.GortPairInfo,
) {
	for key, pair := range pairs {
		if pair == nil || gortSignalKey(pair.Gid1, pair.Gid2) != signalKey {
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

// addInferredPairLocked adds a candidate unless its unordered gid pair has
// already been covered or declared infeasible. Existing suspects are upgraded
// in place so their call-location key remains stable.
func (p *CorpusGort) addInferredPairLocked(gid1, gid2 uint64, confidence float64, sourceType string) bool {
	if gid1 == 0 || gid2 == 0 || gid1 == gid2 {
		return false
	}

	signalKey := gortSignalKey(gid1, gid2)
	if hasGortSignalPair(p.CoveredConPairs, signalKey) || hasGortSignalPair(p.InfeasiblePairs, signalKey) {
		return false
	}

	found := false
	changed := false
	for _, pair := range p.SusConPairs {
		if pair == nil || gortSignalKey(pair.Gid1, pair.Gid2) != signalKey {
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

	if gid2 < gid1 {
		gid1, gid2 = gid2, gid1
	}
	pair := &feedback.GortPairInfo{
		Gid1:       gid1,
		Gid2:       gid2,
		Confidence: confidence,
		SourceType: sourceType,
		IsObserved: false,
	}
	p.SusConPairs[feedback.GortPairKey(pair)] = pair
	return true
}

func (p *CorpusGort) inferFromAnchorLocked(anchor *feedback.GortPairInfo) int {
	if anchor == nil || anchor.Gid1 == 0 || anchor.Gid2 == 0 || anchor.Gid1 == anchor.Gid2 {
		return 0
	}

	changed := 0
	inferSide := func(gid, other uint64) {
		for parent := range p.parents[gid] {
			if p.addInferredPairLocked(parent, other, 0.5, "fuzz_inferred_adjacent") {
				changed++
			}
			for sibling := range p.children[parent] {
				if sibling == gid {
					continue
				}
				if p.addInferredPairLocked(sibling, other, 0.5, "fuzz_inferred_sibling") {
					changed++
				}
			}
		}
		for child := range p.children[gid] {
			if p.addInferredPairLocked(child, other, 0.3, "fuzz_inferred_adjacent") {
				changed++
			}
		}
	}

	inferSide(anchor.Gid1, anchor.Gid2)
	inferSide(anchor.Gid2, anchor.Gid1)
	return changed
}

func (p *CorpusGort) inferFromAllCoveredLocked() int {
	changed := 0
	for _, pair := range p.CoveredConPairs {
		changed += p.inferFromAnchorLocked(pair)
	}
	return changed
}

// ApplySignals 应用goroutine调度有效性信号
// 返回本轮新覆盖的 goroutine 对（从 SusConPairs → CoveredConPairs）
func (p *CorpusGort) ApplySignals(signals []*feedback.CoverageSignal) []*feedback.GortPairInfo {
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
	var newlyCovered []*feedback.GortPairInfo
	for key, pair := range intersection {
		pair.Confidence = 1.0
		pair.IsObserved = true
		pair.SourceType = "fuzz_verified"
		signalKey := gortSignalKey(pair.Gid1, pair.Gid2)
		removeGortSignalPairs(p.SusConPairs, signalKey, p.pairTimeouts, p.TryPairs)
		removeGortSignalPairs(p.InfeasiblePairs, signalKey, nil, nil)
		p.CoveredConPairs[key] = pair
		newlyCovered = append(newlyCovered, pair)
	}
	for _, pair := range newlyCovered {
		p.inferFromAnchorLocked(pair)
	}

	p.RefillTryPairs()
	return newlyCovered
}
