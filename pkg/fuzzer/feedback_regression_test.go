package fuzzer

import (
	"testing"
	"toolkit/pkg/feedback"
)

func TestDelayedPairFeedbackSurvivesRefill(t *testing.T) {
	for _, mode := range []GranularityMode{ModeGoroutine, ModeFunction} {
		t.Run(string(mode), func(t *testing.T) {
			phase := uint32(1)
			corpus := GetAdapter(mode).NewCorpus(&phase)
			var pairs []feedback.ConcurrencyPair
			if mode == ModeGoroutine {
				pairs = []feedback.ConcurrencyPair{&feedback.GortPairInfo{Gid1: 1, Gid2: 2, Confidence: 1}, &feedback.GortPairInfo{Gid1: 3, Gid2: 4, Confidence: .1}}
			} else {
				pairs = []feedback.ConcurrencyPair{&feedback.SuspiciousPairInfo{FuncID1: 1, FuncID2: 2, Confidence: 1}, &feedback.SuspiciousPairInfo{FuncID1: 3, FuncID2: 4, Confidence: .1}}
			}
			corpus.AddConcurrencyPairs(pairs)
			// A result can arrive after another worker has replaced TryPairs.
			covered := corpus.ApplyConcurrencySignals([]*feedback.CoverageSignal{{PreID: 3, NextID: 4, Success: true, Kind: feedback.SignalPairCovered}})
			if len(covered) != 1 {
				t.Fatalf("delayed feedback discarded: %v", covered)
			}
		})
	}
}

func TestOpSuccessWinsOverTimeoutAndCountsProgress(t *testing.T) {
	phase := uint32(1)
	c := NewCorpusOp(&phase)
	pair := &feedback.OpPair{Op1: &feedback.OpInfo{OpId: 1}, Op2: &feedback.OpInfo{OpId: 2}}
	key := opPairKey(pair)
	c.SusConPairs[key] = pair
	c.pairTimeouts[key] = opMaxTimeouts - 1
	got := c.ApplySignals([]*feedback.CoverageSignal{
		{PreID: 1, NextID: 2, Kind: feedback.SignalOpTimeout},
		{PreID: 1, NextID: 2, Kind: feedback.SignalOpCovered, Success: true},
	})
	if got != 1 || len(c.InfeasiblePairs) != 0 || len(c.CoveredConPairs) != 1 {
		t.Fatalf("coverage lost to timeout: progress=%d", got)
	}
}

func TestCandidatesWithoutSignalsDoNotStarveSearch(t *testing.T) {
	phase := uint32(1)
	c := NewCorpusGort(&phase)
	c.AddPair([]*feedback.GortPairInfo{
		{Gid1: 1, Gid2: 2, Confidence: 1},
		{Gid1: 3, Gid2: 4, Confidence: .1},
	})
	for i := 0; i < 50; i++ {
		in := c.GetInput()
		for _, pair := range in.Pairs {
			if pair.ID1() == 3 {
				return
			}
		}
	}
	t.Fatal("unreached high-confidence candidate monopolized all executions")
}
