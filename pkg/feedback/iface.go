package feedback

// ConcurrencyPair is the common interface for concurrent entity pairs.
// Implemented by GortPairInfo (goroutine mode) and SuspiciousPairInfo (function mode).
type ConcurrencyPair interface {
	ID1() uint64
	ID2() uint64
	CallLocation1() CallLocationInfo
	CallLocation2() CallLocationInfo
	GetConfidence() float64
	SetConfidence(float64)
	GetSource() string
	SetSource(string)
	GetObserved() bool
	SetObserved(bool)

	// PairKey returns a normalized dedup key (smaller ID first, includes call locations).
	PairKey() string
	// SignalKey returns a normalized signal-matching key (smaller ID first, no locations).
	SignalKey() string
}

// ConcurrencyEdge is the common interface for topology edges.
// Implemented by GortEdge (goroutine parent-child) and FuncEdge (function caller-callee).
type ConcurrencyEdge interface {
	Parent() uint64
	Child() uint64
	GetCount() uint64
}
