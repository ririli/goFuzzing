package fuzzer

import (
	"fmt"
	"sort"
	"sync"
	"sync/atomic"

	"toolkit/pkg/feedback"
)

// CorpusOp 管理从 [FB] 日志解析出的 OpInfo 操作集合
// 按 OpId 去重，按 Gid 与 FuncID 双索引，分别从 goroutine 并发对（gid 链路）
// 与函数并发对（funcID 链路）中匹配危险操作组合
// 对齐 CorpusGort 的四集合模式（CoveredConPairs / SusConPairs / InfeasiblePairs / TryPairs）
// 无独立的预执行阶段：OP 种子在并发对预执行结束后一次性生成，依赖各自的 CoveredConPairs
type CorpusOp struct {
	mu sync.RWMutex

	// 原始 OpInfo 存储（保持现有逻辑）
	ops    map[uint64]*feedback.OpInfo    // OpId -> OpInfo，编译期唯一 ID 去重
	byGid  map[uint64]map[uint64]struct{} // Gid -> OpId 集合，按 goroutine 索引
	byFunc map[uint64]map[uint64]struct{} // FuncID -> OpId 集合，按函数索引（操作归属于栈上全部函数）

	// 四集合 — 存储 OpPair
	CoveredConPairs map[string]*feedback.OpPair // 已验证触发的危险操作对
	SusConPairs     map[string]*feedback.OpPair // 候选操作对（从 goroutine 对生成）
	InfeasiblePairs map[string]*feedback.OpPair // 不可行操作对（超时过多）
	TryPairs        map[string]*feedback.OpPair // 本轮 fuzzing 输入

	pairTimeouts map[string]int // pairKey -> 累计超时次数
	selectNum    int            // 从 SusConPairs 选取数量，初始=1，自适应调整（上限 64）
	generated    bool           // 是否已从 goroutine 对生成过 OP 种子（只生成一次）
	collected    bool           // 预执行阶段是否已成功收集过 OP 信息（只收集一次，避免跨运行 ObjAddr 不一致）

	phase *uint32 // 指向 cfg.Phase，读取预执行/fuzzing 阶段状态
}

const (
	opMaxTimeouts  = 5  // op 对连续超时阈值
	opMaxSelectNum = 64 // selectNum 上限
)

// NewCorpusOp 创建并初始化 CorpusOp
func NewCorpusOp(phase *uint32) *CorpusOp {
	return &CorpusOp{
		ops:             make(map[uint64]*feedback.OpInfo),
		byGid:           make(map[uint64]map[uint64]struct{}),
		byFunc:          make(map[uint64]map[uint64]struct{}),
		CoveredConPairs: make(map[string]*feedback.OpPair),
		SusConPairs:     make(map[string]*feedback.OpPair),
		InfeasiblePairs: make(map[string]*feedback.OpPair),
		TryPairs:        make(map[string]*feedback.OpPair),
		pairTimeouts:    make(map[string]int),
		selectNum:       1,
		generated:       false,
		phase:           phase,
	}
}

// Add 批量添加 OpInfo，按 OpId 自动去重，并建立 Gid 索引
// 预执行阶段只在首次成功收集后跳过后续调用，避免跨运行 ObjAddr 不一致导致 MatchOpPair 失败
func (co *CorpusOp) Add(opInfos []*feedback.OpInfo) {
	co.mu.Lock()
	defer co.mu.Unlock()

	// 预执行阶段只收集一次：首次成功添加 OP 后，后续调用直接跳过
	if atomic.LoadUint32(co.phase) == 0 && co.collected {
		return
	}

	added := false
	for _, op := range opInfos {
		if op == nil {
			continue
		}
		// 按 OpId 去重：已存在则跳过
		if _, exists := co.ops[op.OpId]; exists {
			continue
		}
		co.ops[op.OpId] = op

		// 建立 Gid -> OpId 索引
		if co.byGid[op.Gid] == nil {
			co.byGid[op.Gid] = make(map[uint64]struct{})
		}
		co.byGid[op.Gid][op.OpId] = struct{}{}

		// 建立 FuncID -> OpId 索引（操作归属于函数栈上全部函数，funcID=0 不参与）
		for _, fid := range op.FuncIDs {
			if fid == 0 {
				continue
			}
			if co.byFunc[fid] == nil {
				co.byFunc[fid] = make(map[uint64]struct{})
			}
			co.byFunc[fid][op.OpId] = struct{}{}
		}
		added = true
	}

	// 预执行阶段首次成功添加 OP 后标记已收集
	if atomic.LoadUint32(co.phase) == 0 && added {
		co.collected = true
	}
}

// getOpsByGid 按 Gid 获取该协程下的所有 OpInfo
func (co *CorpusOp) getOpsByGid(gid uint64) []*feedback.OpInfo {
	opIds := co.byGid[gid]
	if len(opIds) == 0 {
		return nil
	}
	result := make([]*feedback.OpInfo, 0, len(opIds))
	for opId := range opIds {
		if op, ok := co.ops[opId]; ok {
			result = append(result, op)
		}
	}
	return result
}

// getOpsByFunc 按 FuncID 获取归属于该函数（含其调用栈外层函数）的所有 OpInfo
func (co *CorpusOp) getOpsByFunc(fid uint64) []*feedback.OpInfo {
	opIds := co.byFunc[fid]
	if len(opIds) == 0 {
		return nil
	}
	result := make([]*feedback.OpInfo, 0, len(opIds))
	for opId := range opIds {
		if op, ok := co.ops[opId]; ok {
			result = append(result, op)
		}
	}
	return result
}

// opPairKey 生成 OpPair 的去重键：Danger + 操作对象地址 + 两个 OpId
func opPairKey(p *feedback.OpPair) string {
	return fmt.Sprintf("%s-%d-%d-%d", p.Danger, p.Op1.ObjAddr, p.Op1.OpId, p.Op2.OpId)
}

// opSignalKey 生成 OP 调度信号的查找键（方向敏感：preId-nextId）
func opSignalKey(preID, nextID uint64) string {
	return fmt.Sprintf("%d-%d", preID, nextID)
}

// generateFromOps 从两组操作生成危险 OpPair（内部辅助函数）
// caller 负责持有 co.mu 写锁
func (co *CorpusOp) generateFromOps(ops1, ops2 []*feedback.OpInfo, seen map[string]struct{}) {
	for _, op1 := range ops1 {
		for _, op2 := range ops2 {
			// 双向匹配：op1→op2 和 op2→op1
			// select 中的操作无 BF 钩子，只能做 Op1（pre），不能做 Op2（next）
			if !op2.IsSelect {
				if opPair := feedback.MatchOpPair(op1, op2); opPair != nil {
					key := opPairKey(opPair)
					if _, exists := seen[key]; !exists {
						seen[key] = struct{}{}
						co.SusConPairs[key] = opPair
					}
				}
			}
			if !op1.IsSelect {
				if opPair := feedback.MatchOpPair(op2, op1); opPair != nil {
					key := opPairKey(opPair)
					if _, exists := seen[key]; !exists {
						seen[key] = struct{}{}
						co.SusConPairs[key] = opPair
					}
				}
			}
		}
	}
}

// generateFromPair 从单个 goroutine 对生成危险 OpPair（gid 链路）
// caller 负责持有 co.mu 写锁
func (co *CorpusOp) generateFromPair(pair *feedback.GortPairInfo, seen map[string]struct{}) {
	co.generateFromOps(co.getOpsByGid(pair.Gid1), co.getOpsByGid(pair.Gid2), seen)
}

// generateFromFuncPair 从单个函数对生成危险 OpPair（funcID 链路）
// 语义对齐 goroutine 链路：操作归属于函数动态范围（调用栈）而非仅函数自身
// caller 负责持有 co.mu 写锁
func (co *CorpusOp) generateFromFuncPair(pair *feedback.SuspiciousPairInfo, seen map[string]struct{}) {
	co.generateFromOps(co.getOpsByFunc(pair.FuncID1), co.getOpsByFunc(pair.FuncID2), seen)
}

// TryEndPreExec goroutine预执行结束时，一次性从 covered goroutine 对生成全部 OP 对
// 应在 monitor 循环中紧跟 corpusGort.TryEndPreExec 之后调用（经由 CorpusGort.OnPreExecEnd）
func (co *CorpusOp) TryEndPreExec(cg *CorpusGort) {
	if atomic.LoadUint32(co.phase) == 0 {
		return
	}
	co.mu.Lock()
	defer co.mu.Unlock()

	if co.generated {
		return
	}

	// 通过 SnapshotCovered 持读锁取快照，避免直接访问 CorpusGort 内部字段
	covered := cg.SnapshotCovered()
	seen := make(map[string]struct{})
	for _, pair := range covered {
		if pair != nil {
			co.generateFromPair(pair, seen)
		}
	}

	co.generated = true
	co.RefillTryPairs()

	fmt.Printf("[OP_PRESTAGE] Generated %d op pairs from %d covered goroutine pairs\n",
		len(co.SusConPairs), len(covered))
}

// TryEndPreExecForFunc 是函数模式下的 TryEndPreExec 对应方法。
// 通过 [FB] 日志中的 fids 字段将操作归属到函数栈，
// 从 covered 函数对生成 OP 种子，与 goroutine 模式逻辑对齐。
func (co *CorpusOp) TryEndPreExecForFunc(cf *CorpusFunc) {
	if atomic.LoadUint32(co.phase) == 0 {
		return
	}
	co.mu.Lock()
	defer co.mu.Unlock()

	if co.generated {
		return
	}

	covered := cf.SnapshotCovered()
	seen := make(map[string]struct{})
	for _, pair := range covered {
		if pair != nil {
			co.generateFromFuncPair(pair, seen)
		}
	}

	co.generated = true
	co.RefillTryPairs()

	fmt.Printf("[OP_PRESTAGE] Generated %d op pairs from %d covered function pairs\n",
		len(co.SusConPairs), len(covered))
}

// Get 返回 TryPairs 作为 InputOpPair
// 预执行阶段（generated=false）返回 nil；fuzzing 阶段返回 TryPairs 副本
func (co *CorpusOp) Get() *feedback.InputOpPair {
	if !co.generated {
		return nil
	}

	co.mu.RLock()
	defer co.mu.RUnlock()

	result := make([]*feedback.OpPair, 0, len(co.TryPairs))
	for _, pair := range co.TryPairs {
		result = append(result, pair)
	}

	return &feedback.InputOpPair{TryPair: result}
}

// OnGortCovered goroutine 对被 COVERED 时，增量生成对应 OP 对
func (co *CorpusOp) OnGortCovered(pairs []*feedback.GortPairInfo) {
	if atomic.LoadUint32(co.phase) == 0 {
		return
	}
	co.mu.Lock()
	defer co.mu.Unlock()

	if !co.generated {
		return
	}

	seen := make(map[string]struct{})
	for _, pair := range pairs {
		if pair == nil {
			continue
		}
		co.generateFromPair(pair, seen)
	}

	if len(seen) > 0 {
		co.RefillTryPairs()
		fmt.Printf("[OP_INCREMENTAL] Generated %d new op pairs from %d newly covered goroutine pairs\n",
			len(seen), len(pairs))
	}
}

// OnFuncCovered 函数对被 COVERED 时，增量生成对应 OP 对（funcID 链路，对标 OnGortCovered）
func (co *CorpusOp) OnFuncCovered(pairs []*feedback.SuspiciousPairInfo) {
	if atomic.LoadUint32(co.phase) == 0 {
		return
	}
	co.mu.Lock()
	defer co.mu.Unlock()

	if !co.generated {
		return
	}

	seen := make(map[string]struct{})
	for _, pair := range pairs {
		if pair == nil {
			continue
		}
		co.generateFromFuncPair(pair, seen)
	}

	if len(seen) > 0 {
		co.RefillTryPairs()
		fmt.Printf("[OP_INCREMENTAL] Generated %d new op pairs from %d newly covered function pairs\n",
			len(seen), len(pairs))
	}
}

// RefillTryPairs 全量替换 TryPairs，从 SusConPairs 按优先级选取 selectNum 个
func (co *CorpusOp) RefillTryPairs() {
	co.TryPairs = make(map[string]*feedback.OpPair)

	if len(co.SusConPairs) == 0 {
		return
	}

	type cand struct {
		key  string
		pair *feedback.OpPair
	}
	candidates := make([]cand, 0, len(co.SusConPairs))
	for key, pair := range co.SusConPairs {
		candidates = append(candidates, cand{key, pair})
	}

	sort.Slice(candidates, func(i, j int) bool {
		return co.opScore(candidates[i].key) > co.opScore(candidates[j].key)
	})

	n := co.selectNum
	if n > len(candidates) {
		n = len(candidates)
	}
	for i := 0; i < n; i++ {
		co.TryPairs[candidates[i].key] = candidates[i].pair
	}
}

// opScore 计算 OP pair 的综合调度优先级
// 所有 OP 种子无置信度区分，统一使用基准分 0.5
func (co *CorpusOp) opScore(key string) float64 {
	if _, ok := co.SusConPairs[key]; !ok {
		return -1
	}
	return 0.5*10 - float64(co.pairTimeouts[key])*2
}

// ApplySignals 应用 OP 级别调度有效性信号
//
// COVERED_OP → SusConPairs → CoveredConPairs（验证成功）
// TIMEOUT_OP → 累计超时 → >= opMaxTimeouts → InfeasiblePairs（淘汰）
// 无 COVERED 但有 TIMEOUT → selectNum 翻倍扩大搜索范围
func (co *CorpusOp) ApplySignals(signals []*feedback.CoverageSignal) {
	co.mu.Lock()
	defer co.mu.Unlock()

	// 分类信号
	coveredSigKeys := make(map[string]struct{})
	timeoutSigKeys := make(map[string]struct{})
	for _, sig := range signals {
		if sig == nil {
			continue
		}
		if sig.Kind != feedback.SignalOpCovered && sig.Kind != feedback.SignalOpTimeout {
			continue
		}
		sk := opSignalKey(sig.PreID, sig.NextID)
		if sig.Success {
			coveredSigKeys[sk] = struct{}{}
		} else {
			timeoutSigKeys[sk] = struct{}{}
		}
	}

	// 超时处理：匹配 TryPairs 中的对，累计超时
	for sk := range timeoutSigKeys {
		for tryKey, tryPair := range co.TryPairs {
			if opSignalKey(tryPair.Op1.OpId, tryPair.Op2.OpId) != sk {
				continue
			}
			co.pairTimeouts[tryKey]++
			if co.pairTimeouts[tryKey] >= opMaxTimeouts {
				co.InfeasiblePairs[tryKey] = tryPair
				delete(co.SusConPairs, tryKey)
				delete(co.pairTimeouts, tryKey)
				fmt.Printf("[op_signal] op pair %s infeasible after %d timeouts\n",
					tryKey, opMaxTimeouts)
			}
		}
	}

	// 计算 COVERED ∩ TryPairs
	intersection := make(map[string]*feedback.OpPair)
	for tryKey, tryPair := range co.TryPairs {
		if _, ok := coveredSigKeys[opSignalKey(tryPair.Op1.OpId, tryPair.Op2.OpId)]; ok {
			intersection[tryKey] = tryPair
		}
	}

	// 无 COVERED 但有 TIMEOUT → 扩大 selectNum
	if len(intersection) == 0 && len(timeoutSigKeys) > 0 {
		co.selectNum *= 2
		if co.selectNum > opMaxSelectNum {
			co.selectNum = opMaxSelectNum
		}
	}

	// COVERED 对：SusConPairs → CoveredConPairs
	for key, pair := range intersection {
		co.CoveredConPairs[key] = pair
		delete(co.SusConPairs, key)
		delete(co.pairTimeouts, key)
		fmt.Printf("[op_signal] op pair %s COVERED → covered\n", key)
	}

	co.RefillTryPairs()
}
