package fuzzer

import (
	"fmt"

	"toolkit/pkg/feedback"
)

// PairCorpus is the unified interface for concurrent pair corpora.
// Both CorpusGort (goroutine mode) and CorpusFunc (function mode) implement this interface,
// allowing the Monitor to operate without mode-specific branching.
type PairCorpus interface {
	// GetInput returns pairs to try in the next execution.
	// Returns nil during pre-execution (phase 0).
	GetInput() *PairInput

	// AddConcurrencyPairs merges pairs parsed from a single execution's stderr.
	AddConcurrencyPairs(pairs []feedback.ConcurrencyPair)

	// AddConcurrencyEdges merges topology edges parsed from a single execution's stderr.
	// Returns the number of newly discovered edges.
	AddConcurrencyEdges(edges []feedback.ConcurrencyEdge) int

	// TryEndPreExec checks whether pre-execution seed collection should end,
	// transitioning to phase 1 (fuzzing) when stable or max rounds reached.
	TryEndPreExec(maxRounds int)

	// ApplyConcurrencySignals applies coverage/timeout signals from an execution.
	// Returns newly covered pairs for OP corpus integration.
	ApplyConcurrencySignals(signals []*feedback.CoverageSignal) []feedback.ConcurrencyPair

	// OnCoveredByOp is called when goroutine pairs are newly covered,
	// allowing the OP corpus to generate corresponding operation pairs.
	// No-op in function mode.
	OnCoveredByOp(opCorpus *CorpusOp, pairs []feedback.ConcurrencyPair)

	// InPreExec returns true during pre-execution seed collection (phase 0).
	InPreExec() bool

	// ModeName returns a human-readable mode name for logging.
	ModeName() string
}

// PairInput represents the input passed to the test binary via the Input environment variable.
// It unifies goroutine pairs (GortPairInfo) and function pairs (SuspiciousPairInfo).
type PairInput struct {
	Pairs []feedback.ConcurrencyPair
}

// ToString serializes pairs for the Input environment variable.
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

// IsEmpty returns true if there are no pairs.
func (pi *PairInput) IsEmpty() bool {
	return pi == nil || len(pi.Pairs) == 0
}

// adaptConcurrencyPairs converts []ConcurrencyPair to []*GortPairInfo.
// Returns nil if any pair is not a GortPairInfo. Used by CorpusGort.
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

// toFuncPairs converts []ConcurrencyPair to []*SuspiciousPairInfo.
// Returns nil if any pair is not a SuspiciousPairInfo. Used by CorpusFunc.
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

// toConcurrencyPairs converts []*GortPairInfo to []ConcurrencyPair.
func gortToConcurrencyPairs(gortPairs []*feedback.GortPairInfo) []feedback.ConcurrencyPair {
	result := make([]feedback.ConcurrencyPair, len(gortPairs))
	for i, p := range gortPairs {
		result[i] = p
	}
	return result
}

// funcToConcurrencyPairs converts []*SuspiciousPairInfo to []ConcurrencyPair.
func funcToConcurrencyPairs(funcPairs []*feedback.SuspiciousPairInfo) []feedback.ConcurrencyPair {
	result := make([]feedback.ConcurrencyPair, len(funcPairs))
	for i, p := range funcPairs {
		result[i] = p
	}
	return result
}
