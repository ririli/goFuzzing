package fuzzer

import "toolkit/pkg/feedback"

type CorpusPair struct {
	CoveredConPairs []*feedback.SuspiciousPairInfo
	SusConPairs     []*feedback.SuspiciousPairInfo
}

func (p *CorpusPair) NewCorpusPair() {
	p.SusConPairs = make([]*feedback.SuspiciousPairInfo, 0)
	p.CoveredConPairs = make([]*feedback.SuspiciousPairInfo, 0)
}
