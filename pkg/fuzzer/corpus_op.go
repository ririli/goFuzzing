package fuzzer

import (
	"fmt"
	"sync"

	"toolkit/pkg/feedback"
)

// CorpusOp 管理从 [FB] 日志解析出的 OpInfo 操作集合
// 按 OpId 去重，按 FuncId 索引，支持从并发对中匹配危险操作组合
type CorpusOp struct {
	mu     sync.RWMutex
	ops    map[uint64]*feedback.OpInfo    // OpId -> OpInfo，编译期唯一 ID 去重
	byFunc map[uint64]map[uint64]struct{} // FuncId -> OpId 集合，按函数索引

	// blockedPairs 记录已处理/屏蔽的操作对
	// key: opPairKey, value: -1=已完成(panic触发过), >0=超时次数(>=阈值=永久屏蔽)
	blockedPairs map[string]int
}

// NewCorpusOp 创建并初始化 CorpusOp
func NewCorpusOp() *CorpusOp {
	return &CorpusOp{
		ops:          make(map[uint64]*feedback.OpInfo),
		byFunc:       make(map[uint64]map[uint64]struct{}),
		blockedPairs: make(map[string]int),
	}
}

// Add 批量添加 OpInfo，按 OpId 自动去重，并建立 FuncId 索引
func (co *CorpusOp) Add(opInfos []*feedback.OpInfo) {
	co.mu.Lock()
	defer co.mu.Unlock()

	for _, op := range opInfos {
		if op == nil {
			continue
		}
		// 按 OpId 去重：已存在则跳过
		if _, exists := co.ops[op.OpId]; exists {
			continue
		}
		co.ops[op.OpId] = op

		// 建立 FuncId -> OpId 索引
		if co.byFunc[op.FuncId] == nil {
			co.byFunc[op.FuncId] = make(map[uint64]struct{})
		}
		co.byFunc[op.FuncId][op.OpId] = struct{}{}
	}
}

// Get 从 CorpusPair 的 CoveredConPairs 中提取并发函数对，
// 匹配每对函数各自包含的 OpInfo 操作，找出可构成危险 OpPair 的组合
func (co *CorpusOp) Get(cp *CorpusPair) *feedback.InputOpPair {
	co.mu.RLock()
	defer co.mu.RUnlock()

	var results []*feedback.OpPair
	seen := make(map[string]struct{}) // 结果去重

	for _, pair := range cp.CoveredConPairs {
		if pair == nil {
			continue
		}
		ops1 := co.getOpsByFuncId(pair.FuncID1)
		ops2 := co.getOpsByFuncId(pair.FuncID2)

		for _, op1 := range ops1 {
			for _, op2 := range ops2 {
				// 尝试两个方向的匹配（顺序决定 Danger 类型语义）
				// 注意：select 中的操作无 BF 钩子，只能做 Op1（pre），不能做 Op2（next）
				if !op2.IsSelect {
					if opPair := feedback.MatchOpPair(op1, op2); opPair != nil {
						key := opPairKey(opPair)
						if _, exists := seen[key]; !exists {
							if co.blockedPairs[key] > 0 {
								continue
							}
							seen[key] = struct{}{}
							results = append(results, opPair)
						}
					}
				}
				if !op1.IsSelect {
					if opPair := feedback.MatchOpPair(op2, op1); opPair != nil {
						key := opPairKey(opPair)
						if _, exists := seen[key]; !exists {
							if co.blockedPairs[key] > 0 {
								continue
							}
							seen[key] = struct{}{}
							results = append(results, opPair)
						}
					}
				}
			}
		}
	}
	return &feedback.InputOpPair{TryPair: results}
}

// getOpsByFuncId 按 FuncId 获取该函数下的所有 OpInfo
func (co *CorpusOp) getOpsByFuncId(funcId uint64) []*feedback.OpInfo {
	opIds := co.byFunc[funcId]
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

const maxOpTimeouts = 5 // op 对连续超时阈值

// ApplySignals 应用 op 级别调度有效性信号
//
// COVERED_OP → 危险操作已触发 panic → blockedPairs[key] = -1（永久屏蔽）
// TIMEOUT_OP → 累计超时次数 → >= maxOpTimeouts 则屏蔽
func (co *CorpusOp) ApplySignals(signals []*feedback.CoverageSignal) {
	co.mu.Lock()
	defer co.mu.Unlock()

	for _, sig := range signals {
		if sig == nil {
			continue
		}
		// 只处理 op 级别信号
		if sig.Kind != feedback.SignalOpCovered && sig.Kind != feedback.SignalOpTimeout {
			continue
		}

		pairKey := co.findPairKey(sig.PreID, sig.NextID)
		if pairKey == "" {
			continue
		}

		if sig.Success {
			// COVERED_OP → 危险操作已触发 → 永久屏蔽
			co.blockedPairs[pairKey] = -1
			fmt.Printf("[op_signal] op pair %s COVERED → blocked (panic triggered)\n", pairKey)
		} else {
			// TIMEOUT_OP → 累计超时
			if co.blockedPairs[pairKey] == -1 {
				continue // 已完成的跳过
			}
			co.blockedPairs[pairKey]++
			if co.blockedPairs[pairKey] >= maxOpTimeouts {
				fmt.Printf("[op_signal] op pair %s removed after %d timeouts\n", pairKey, maxOpTimeouts)
			}
		}
	}
}

// findPairKey 根据信号中的 (preId, nextId) 构建 opPairKey
// 利用 ops map 查找对应的 OpInfo，通过 MatchOpPair 推导 Danger 类型
func (co *CorpusOp) findPairKey(preId, nextId uint64) string {
	op1, ok1 := co.ops[preId]
	op2, ok2 := co.ops[nextId]
	if !ok1 || !ok2 {
		return ""
	}
	pair := feedback.MatchOpPair(op1, op2)
	if pair == nil {
		return ""
	}
	return opPairKey(pair)
}
