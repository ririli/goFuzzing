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
}

// NewCorpusOp 创建并初始化 CorpusOp
func NewCorpusOp() *CorpusOp {
	return &CorpusOp{
		ops:    make(map[uint64]*feedback.OpInfo),
		byFunc: make(map[uint64]map[uint64]struct{}),
	}
}

// Add 批量添加 OpInfo，按 OpId 自动去重，并建立 FuncId 索引
func (co *CorpusOp) Add(opInfos []*feedback.OpInfo) {

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
				if opPair := feedback.MatchOpPair(op1, op2); opPair != nil {
					key := opPairKey(opPair)
					if _, exists := seen[key]; !exists {
						seen[key] = struct{}{}
						results = append(results, opPair)
					}
				}
				if opPair := feedback.MatchOpPair(op2, op1); opPair != nil {
					key := opPairKey(opPair)
					if _, exists := seen[key]; !exists {
						seen[key] = struct{}{}
						results = append(results, opPair)
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
