package fuzzer

import "toolkit/pkg/feedback"

// ConcurrentPair 是并发实体对的公共接口。
// goroutine 对（GortPairInfo）与函数对（SuspiciousPairInfo）
// 均通过其访问器方法实现该接口。
type ConcurrentPair = feedback.ConcurrencyPair

// TopologyEdge 是拓扑边的公共接口。
type TopologyEdge = feedback.ConcurrencyEdge

// PairParser 将 stderr 文本解析为 ConcurrentPairs 与 OpInfos。
type PairParser func(stderr string) ([]ConcurrentPair, []*feedback.OpInfo, error)

// EdgeParser 将 stderr 文本解析为 TopologyEdges。
type EdgeParser func(stderr string) ([]TopologyEdge, error)

// GranularityAdapter 为 Monitor 聚合模式相关操作。
// 这是在 goroutine 级与函数级 fuzzing 颗粒度之间切换的主要抽象。
// 模式选择（parser 与语料库创建）必须仅通过该 adapter 进行。
type GranularityAdapter struct {
	Mode       GranularityMode
	ParsePairs PairParser
	ParseEdges EdgeParser
}

// NewCorpus 创建与该 adapter 模式匹配的 PairCorpus 实现。
func (a *GranularityAdapter) NewCorpus(phase *uint32) PairCorpus {
	if a.Mode == ModeFunction {
		return NewCorpusFunc(phase)
	}
	return NewCorpusGort(phase)
}

// GetAdapter 返回给定模式的 GranularityAdapter。
func GetAdapter(mode GranularityMode) *GranularityAdapter {
	switch mode {
	case ModeFunction:
		return functionAdapter()
	default:
		return goroutineAdapter()
	}
}

func goroutineAdapter() *GranularityAdapter {
	return &GranularityAdapter{
		Mode:       ModeGoroutine,
		ParsePairs: wrapGortPairParser,
		ParseEdges: wrapGortEdgeParser,
	}
}

func functionAdapter() *GranularityAdapter {
	return &GranularityAdapter{
		Mode:       ModeFunction,
		ParsePairs: wrapFuncPairParser,
		ParseEdges: wrapFuncEdgeParser,
	}
}

// --- 解析器包装：将具体解析函数适配为 PairParser/EdgeParser 签名 ---

func wrapGortPairParser(stderr string) ([]ConcurrentPair, []*feedback.OpInfo, error) {
	gortPairs, ops, err := feedback.ParseGortPairs(stderr)
	if err != nil {
		return nil, nil, err
	}
	pairs := make([]ConcurrentPair, len(gortPairs))
	for i, p := range gortPairs {
		pairs[i] = p
	}
	return pairs, ops, nil
}

func wrapGortEdgeParser(stderr string) ([]TopologyEdge, error) {
	gortEdges, err := feedback.ParseGortEdges(stderr)
	if err != nil {
		return nil, err
	}
	edges := make([]TopologyEdge, len(gortEdges))
	for i, e := range gortEdges {
		edges[i] = e
	}
	return edges, nil
}

func wrapFuncPairParser(stderr string) ([]ConcurrentPair, []*feedback.OpInfo, error) {
	funcPairs, ops, err := feedback.ParseStdPairs(stderr)
	if err != nil {
		return nil, nil, err
	}
	pairs := make([]ConcurrentPair, len(funcPairs))
	for i, p := range funcPairs {
		pairs[i] = p
	}
	return pairs, ops, nil
}

func wrapFuncEdgeParser(stderr string) ([]TopologyEdge, error) {
	funcEdges, err := feedback.ParseFuncEdges(stderr)
	if err != nil {
		return nil, err
	}
	edges := make([]TopologyEdge, len(funcEdges))
	for i, e := range funcEdges {
		edges[i] = e
	}
	return edges, nil
}
