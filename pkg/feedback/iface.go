package feedback

// ConcurrencyPair 是并发实体对的公共接口。
// 由 GortPairInfo（goroutine 模式）和 SuspiciousPairInfo（function 模式）实现。
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

	// PairKey 返回归一化的去重键（较小 ID 在前，包含调用位置）。
	PairKey() string
	// SignalKey 返回归一化的信号匹配键（较小 ID 在前，不含位置）。
	SignalKey() string
}

// ConcurrencyEdge 是拓扑边的公共接口。
// 由 GortEdge（goroutine 父子关系）和 FuncEdge（函数调用者-被调用者关系）实现。
type ConcurrencyEdge interface {
	Parent() uint64
	Child() uint64
	GetCount() uint64
}
