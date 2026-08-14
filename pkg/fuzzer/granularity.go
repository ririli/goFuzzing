package fuzzer

import "toolkit/pkg/feedback"

// ConcurrentPair is the common interface for pairs of concurrent entities.
// Both goroutine pairs (GortPairInfo) and function pairs (SuspiciousPairInfo)
// implement this interface via their accessor methods.
type ConcurrentPair = feedback.ConcurrencyPair

// TopologyEdge is the common interface for topology edges.
type TopologyEdge = feedback.ConcurrencyEdge

// PairParser parses stderr text into ConcurrentPairs and OpInfos.
type PairParser func(stderr string) ([]ConcurrentPair, []*feedback.OpInfo, error)

// EdgeParser parses stderr text into TopologyEdges.
type EdgeParser func(stderr string) ([]TopologyEdge, error)

// GranularityAdapter bundles mode-specific operations for the Monitor.
// This is the primary abstraction for switching between goroutine-level
// and function-level fuzzing granularities. Mode selection (parser and
// corpus creation) must go through this adapter only.
type GranularityAdapter struct {
	Mode       GranularityMode
	ParsePairs PairParser
	ParseEdges EdgeParser
}

// NewCorpus creates the PairCorpus implementation matching this adapter's mode.
func (a *GranularityAdapter) NewCorpus(phase *uint32) PairCorpus {
	if a.Mode == ModeFunction {
		return NewCorpusFunc(phase)
	}
	return NewCorpusGort(phase)
}

// GetAdapter returns the GranularityAdapter for the given mode.
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

// --- Parser wrappers: adapt concrete parser functions to the PairParser/EdgeParser signatures ---

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
