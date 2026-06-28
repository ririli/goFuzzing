package breakpoint

// Strategy 定义断点控制策略的接口。
// 单栏（Config）和双栏（BarrierConfig）均实现此接口，
// 可通过环境变量 BARRIER_MODE=double 在运行时切换。
type Strategy interface {
	// PointControl 在函数入口处执行断点控制，决定放行或阻塞
	PointControl(funcId uint64)
	// ParseInput 从环境变量 Input 解析可疑函数对并初始化内部状态
	ParseInput()
	// ParseSusPairs 解析字符串格式的可疑函数对
	ParseSusPairs(s string)
	// HasActive 返回是否有活跃的断点配置
	HasActive() bool
}
