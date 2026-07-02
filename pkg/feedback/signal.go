package feedback

// SignalKind 调度信号种类
type SignalKind string

const (
	SignalGortCovered SignalKind = "gort_covered" // goroutine对调度成功
	SignalGortTimeout SignalKind = "gort_timeout" // goroutine对调度超时
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
