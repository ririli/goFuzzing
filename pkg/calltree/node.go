// Package calltree 提供函数调用树的数据模型与收集器。
// 它是 callstack 拆分的核心子包，不依赖 overlap / breakpoint / callstack。
package calltree

import "fmt"

// FunctionCallNode 表示函数调用树中的一个节点
type FunctionCallNode struct {
	FuncID        uint64              // 函数的唯一id，插桩时确认
	CallID        uint64              // 调用唯一ID
	FuncName      string              // 函数名
	CallLoc       CallLocation        // 调用位置信息（文件名、行号、调用者函数）
	GoroutineID   int                 // 所在的goroutine ID
	StartUnixNano int64               // 函数开始时间
	EndUnixNano   int64               // 函数结束时间（调用结束时设置）
	Parent        *FunctionCallNode   // 父节点（调用者）
	Children      []*FunctionCallNode // 子节点（被调用的函数）
	Depth         int                 // 调用深度
}

// CallLocation 记录函数调用的具体位置
type CallLocation struct {
	File     string  // 调用发生的源文件路径
	Line     int     // 调用发生的行号
	FuncName string  // 调用者的函数名
	PC       uintptr // 程序计数器地址（用于调试）
}

// String 返回调用位置的字符串表示
func (cl *CallLocation) String() string {
	return fmt.Sprintf("%s:%d in %s", cl.File, cl.Line, cl.FuncName)
}
