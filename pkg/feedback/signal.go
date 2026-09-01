package feedback

// SignalKind 调度信号种类
type SignalKind string

const (
	// SignalPairCovered/SignalPairTimeout 为中性的并发对信号种类，
	// goroutine 与 function 两种粒度共用 {COVERED}/{TIMEOUT} 前缀。
	SignalPairCovered SignalKind = "pair_covered" // 并发对调度成功
	SignalPairTimeout SignalKind = "pair_timeout" // 并发对调度超时
	SignalOpCovered   SignalKind = "op_covered"   // 操作对调度成功
	SignalOpTimeout   SignalKind = "op_timeout"   // 操作对调度超时
)

// CoverageSignal 调度有效性信号（从 stdout 解析，无需调用栈）
type CoverageSignal struct {
	PreID   uint64     // 前置 ID
	NextID  uint64     // 后置 ID
	Success bool       // true=covered成功, false=timeout超时
	Kind    SignalKind // 信号种类: gort / op
}
