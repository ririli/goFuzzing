package callstack

// SuspiciousConcurrentPair 表示一个可疑的并发对
type SuspiciousConcurrentPair struct {
	FuncID1    uint64 // 插桩的函数ID
	FuncID2    uint64 // 插桩的函数ID
	CallID1    uint64 // 第一个函数的调用ID
	CallID2    uint64 // 第二个函数的调用ID
	FuncName1  string // 第一个函数名
	FuncName2  string // 第二个函数名
	Goroutine1 int    // 第一个函数所在的goroutine ID
	Goroutine2 int    // 第二个函数所在的goroutine ID
	Reason     string // 推断原因（基于哪个重叠对推断出来的）
}
