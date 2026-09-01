package fuzzer

import (
	"fmt"

	"toolkit/pkg/feedback"
)

// PairCorpus 是并发对语料库的统一接口。
// CorpusGort（goroutine 模式）与 CorpusFunc（function 模式）都实现了该接口，
// 使 Monitor 无需按模式分支即可工作。
type PairCorpus interface {
	// GetInput 返回下一次执行中要尝试的配对。
	// 预执行阶段（phase 0）返回 nil。
	GetInput() *PairInput

	// AddConcurrencyPairs 合并单次执行 stderr 中解析出的配对。
	AddConcurrencyPairs(pairs []feedback.ConcurrencyPair)

	// AddConcurrencyEdges 合并单次执行 stderr 中解析出的拓扑边。
	// 返回新发现的边数。
	AddConcurrencyEdges(edges []feedback.ConcurrencyEdge) int

	// TryEndPreExec 判断预执行种子收集是否应结束，
	// 达到稳定条件或最大轮次时转换到 phase 1（fuzzing）。
	TryEndPreExec(maxRounds int)

	// OnPreExecEnd 将 OP corpus 接入预执行→fuzzing 转换。
	// 必须在 TryEndPreExec 之后立即调用。goroutine 模式从已覆盖的
	// goroutine 对生成 OP 对；function 模式跳过生成。
	OnPreExecEnd(opCorpus *CorpusOp)

	// ApplyConcurrencySignals 应用单次执行的覆盖/超时信号。
	// 返回新覆盖的配对，用于 OP corpus 集成。
	ApplyConcurrencySignals(signals []*feedback.CoverageSignal) []feedback.ConcurrencyPair

	// OnCoveredByOp 在 goroutine 对被新覆盖时被调用，
	// 允许 OP corpus 生成对应的操作对。
	// function 模式下无操作。
	OnCoveredByOp(opCorpus *CorpusOp, pairs []feedback.ConcurrencyPair)

	// InPreExec 在预执行种子收集阶段（phase 0）返回 true。
	InPreExec() bool

	// ModeName 返回用于日志的可读模式名称。
	ModeName() string
}

// PairInput 表示通过 Input 环境变量传给测试二进制的输入。
// 它统一了 goroutine 对（GortPairInfo）与函数对（SuspiciousPairInfo）。
type PairInput struct {
	Pairs []feedback.ConcurrencyPair
}

// pairSignalKey 生成方向无关的信号匹配键（较小 ID 在前），
// goroutine 与 function 两种粒度通用。
func pairSignalKey(id1, id2 uint64) string {
	if id1 <= id2 {
		return fmt.Sprintf("%d-%d", id1, id2)
	}
	return fmt.Sprintf("%d-%d", id2, id1)
}

// ToString 将配对序列化为 Input 环境变量的输入。
// Format: (id1,id2)(id3,id4)...
func (pi *PairInput) ToString() string {
	if pi == nil || len(pi.Pairs) == 0 {
		return ""
	}
	var result string
	for _, p := range pi.Pairs {
		result += fmt.Sprintf("(%d,%d)", p.ID1(), p.ID2())
	}
	return result
}

// IsEmpty 在没有配对时返回 true。
func (pi *PairInput) IsEmpty() bool {
	return pi == nil || len(pi.Pairs) == 0
}

// toGortPairs 将 []ConcurrencyPair 转换为 []*GortPairInfo。
// 非 GortPairInfo 的配对会被忽略。由 CorpusGort 使用。
func toGortPairs(pairs []feedback.ConcurrencyPair) []*feedback.GortPairInfo {
	if len(pairs) == 0 {
		return nil
	}
	result := make([]*feedback.GortPairInfo, 0, len(pairs))
	for _, p := range pairs {
		if gp, ok := p.(*feedback.GortPairInfo); ok {
			result = append(result, gp)
		}
	}
	return result
}

// toFuncPairs 将 []ConcurrencyPair 转换为 []*SuspiciousPairInfo。
// 非 SuspiciousPairInfo 的配对会被忽略。由 CorpusFunc 使用。
func toFuncPairs(pairs []feedback.ConcurrencyPair) []*feedback.SuspiciousPairInfo {
	if len(pairs) == 0 {
		return nil
	}
	result := make([]*feedback.SuspiciousPairInfo, 0, len(pairs))
	for _, p := range pairs {
		if fp, ok := p.(*feedback.SuspiciousPairInfo); ok {
			result = append(result, fp)
		}
	}
	return result
}

// gortToConcurrencyPairs 将 []*GortPairInfo 转换为 []ConcurrencyPair。
func gortToConcurrencyPairs(gortPairs []*feedback.GortPairInfo) []feedback.ConcurrencyPair {
	result := make([]feedback.ConcurrencyPair, len(gortPairs))
	for i, p := range gortPairs {
		result[i] = p
	}
	return result
}

// funcToConcurrencyPairs 将 []*SuspiciousPairInfo 转换为 []ConcurrencyPair。
func funcToConcurrencyPairs(funcPairs []*feedback.SuspiciousPairInfo) []feedback.ConcurrencyPair {
	result := make([]feedback.ConcurrencyPair, len(funcPairs))
	for i, p := range funcPairs {
		result[i] = p
	}
	return result
}
