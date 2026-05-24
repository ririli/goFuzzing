package callstack

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

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

// CallStackCollector 调用栈收集器
type CallStackCollector struct {
	mu           sync.RWMutex
	callTrees    map[int]*FunctionCallNode    // goroutineID -> 当前调用树的活动根节点
	nodePool     map[uint64]*FunctionCallNode // 所有节点的全局池（按CallID索引）
	funcIndex    map[uint64][]uint64          // FuncID -> CallID列表（目前未使用，预留）
	nextCallID   uint64
	callStackMap map[int][]*FunctionCallNode // goroutineID -> 当前调用栈（用于快速回溯）
}

// NewCallStackCollector 创建新的调用栈收集器
func NewCallStackCollector() *CallStackCollector {
	return &CallStackCollector{
		callTrees:    make(map[int]*FunctionCallNode),
		nodePool:     make(map[uint64]*FunctionCallNode),
		funcIndex:    make(map[uint64][]uint64),
		callStackMap: make(map[int][]*FunctionCallNode),
		nextCallID:   1,
	}
}

// getCurrentGoroutineID 获取当前goroutine ID（性能优化版本）
func getCurrentGoroutineID() int {
	buf := make([]byte, 64)
	n := runtime.Stack(buf, false)
	var id int
	// 快速解析 "goroutine 18 [running]:" 中的 18
	if n > 0 {
		fmt.Sscanf(string(buf[:n]), "goroutine %d", &id)
	}
	return id
}

// getCallInfo 记录被插桩函数的name以及其被调用位置信息
func getCallInfo() (funcName string, callLoc CallLocation) {
	// 获取当前函数(被插桩函数)的信息
	// level=3: runtime.Caller -> getCallInfo -> EnterFunction -> Trace -> defer闭包 -> 目标函数
	level := 3
	pc, selfFile, selfLine, ok := runtime.Caller(level)
	if !ok {
		return "unknown", CallLocation{
			File:     "unknown",
			Line:     0,
			FuncName: "unknown",
			PC:       0,
		}
	}

	// 获取当前函数名
	fn := runtime.FuncForPC(pc)
	if fn != nil {
		funcName = fn.Name()
	} else {
		funcName = "unknown"
	}

	// 尝试获取调用者信息
	callerPC, callerFile, callerLine, ok := runtime.Caller(level + 1)

	var finalFile string
	var finalLine int
	var tag string // 用于标记 (Test) 或 (go)

	if ok {
		// 检查调用者是否是标准库
		callerFn := runtime.FuncForPC(callerPC)
		if callerFn != nil {
			// 判断是否是标准库或 runtime（通过文件路径判断）
			if isStandardLibrary(callerFile) {
				// 调用者在标准库中，使用被插桩函数自身的位置
				finalFile = selfFile
				finalLine = selfLine
				//fmt.Println(callerFile)
				// 添加标记区分 Test 函数和 go 关键字启动的函数
				if strings.Contains(callerFile, "src/testing") {
					tag = " (Test)"
				} else if strings.Contains(callerFile, "src/runtime") {
					tag = " (go)"
				}
				finalFile += tag
			} else {
				// 调用者是用户代码，使用调用者的位置
				finalFile = callerFile
				finalLine = callerLine
			}
		}
		// 将绝对路径转换为相对路径
		relativeFile := convertToRelativePath(finalFile)

		// 记录被插桩函数的位置信息
		callLoc = CallLocation{
			File:     relativeFile,
			Line:     finalLine,
			FuncName: callerFn.Name(),
			PC:       pc,
		}
		return funcName, callLoc
	} else {
		return "unknown", CallLocation{
			File:     "unknown",
			Line:     0,
			FuncName: "unknown",
			PC:       0,
		}
	}
}

// convertToRelativePath 将绝对路径转换为相对路径
// 例如：D:\Program Files\goProjects\src\gopie\testdata\myTest\instFunc_test.go
// 转换为：gopie/testdata/myTest/instFunc_test.go
func convertToRelativePath(absPath string) string {
	// 查找 "gopie" 关键字的位置
	// 支持 Windows 和 Unix 路径分隔符
	keyDir := "gopie"

	// 尝试找到 gopie 目录的位置
	idx := strings.Index(absPath, keyDir)
	if idx == -1 {
		// 如果找不到 gopie，返回原始路径
		return absPath
	}

	// 从 gopie 开始截取路径
	relativePath := absPath[idx:]

	// 统一使用正斜杠
	relativePath = filepath.ToSlash(relativePath)

	return relativePath
}

// isStandardLibrary 判断是否是标准库或 runtime 的代码
func isStandardLibrary(filePath string) bool {
	// 常见的标准库前缀
	standardPrefixes := []string{
		"runtime.",
		"internal/",
		"sync.",
		"time.",
		"testing.",
		"fmt.",
		"os.",
		"io.",
		"context.",
		"reflect.",
	}

	for _, prefix := range standardPrefixes {
		if len(filePath) >= len(prefix) && filePath[:len(prefix)] == prefix {
			return true
		}
	}

	// 检查路径中是否包含标准库的特征路径
	standardPaths := []string{
		"/src/runtime/",
		"/src/internal/",
		"/src/testing/",
		"/src/sync/",
		"/src/time/",
		"\\src\\runtime\\",
		"\\src\\internal\\",
		"\\src\\testing\\",
		"\\src\\sync\\",
		"\\src\\time\\",
	}

	for _, path := range standardPaths {
		if strings.Contains(filePath, path) {
			return true
		}
	}

	return false
}

//// 获取当前函数名（跳过指定层数）
//func getCurrentFuncName(skip int) string {
//	pc, _, _, ok := runtime.Caller(skip)
//	if !ok {
//		return "unknown"
//	}
//	fn := runtime.FuncForPC(pc)
//	if fn == nil {
//		return "unknown"
//	}
//
//	//// 提取简化的函数名（去掉包路径）
//	//fullName := fn.Name()
//	//parts := strings.Split(fullName, ".")
//	//if len(parts) > 0 {
//	//	return parts[len(parts)-1]
//	//}
//	return fn.Name()
//}

// EnterFunction 记录函数进入
func (c *CallStackCollector) EnterFunction(funcID uint64) *FunctionCallNode {
	goroutineID := getCurrentGoroutineID()
	funcName, callLoc := getCallInfo()
	c.mu.Lock()
	defer c.mu.Unlock()

	// 创建新节点
	node := &FunctionCallNode{
		FuncID:        funcID,
		CallID:        c.nextCallID,
		FuncName:      funcName,
		GoroutineID:   goroutineID,
		StartUnixNano: time.Now().UnixNano(),
		Depth:         0,
		CallLoc:       callLoc,
	}
	c.nextCallID++
	// 保存到节点池
	c.nodePool[node.CallID] = node

	c.funcIndex[funcID] = append(c.funcIndex[funcID], node.CallID)

	// 获取当前Goroutine的调用栈
	stack := c.callStackMap[goroutineID]

	if len(stack) == 0 {
		// 这是该goroutine的根调用
		c.callTrees[goroutineID] = node
	} else {
		// 获取父节点（调用栈顶）
		parent := stack[len(stack)-1]
		node.Parent = parent
		node.Depth = parent.Depth + 1
		parent.Children = append(parent.Children, node)
	}

	// 压入调用栈
	c.callStackMap[goroutineID] = append(stack, node)

	return node
}

// ExitFunction 记录函数退出
func (c *CallStackCollector) ExitFunction(node *FunctionCallNode) {
	if node == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	node.EndUnixNano = time.Now().UnixNano()
	goroutineID := node.GoroutineID

	// 从调用栈中弹出
	if stack, exists := c.callStackMap[goroutineID]; exists && len(stack) > 0 {
		// 验证栈顶确实是当前节点
		if stack[len(stack)-1].CallID == node.CallID {
			// 弹出栈顶
			c.callStackMap[goroutineID] = stack[:len(stack)-1]
		} else {
			// 栈不一致,可能是异常情况,进行栈清理
			c.cleanupStack(goroutineID, node.CallID)
		}
	}
}

// ClearActiveStacks 清理所有活动调用栈(收集结束后释放内存)
// 注意:此操作不会影响已构建的调用树和节点池
func (c *CallStackCollector) ClearActiveStacks() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.callStackMap = make(map[int][]*FunctionCallNode)
}

// 清理不一致的调用栈
func (c *CallStackCollector) cleanupStack(goroutineID int, callID uint64) {
	stack := c.callStackMap[goroutineID]
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i].CallID == callID {
			c.callStackMap[goroutineID] = stack[:i]
			return
		}
	}
}

// GetCallTree 获取指定goroutine的完整调用树
func (c *CallStackCollector) GetCallTree(goroutineID int) *FunctionCallNode {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.callTrees[goroutineID]
}

// GetAllCallTrees 获取所有goroutine的调用树
func (c *CallStackCollector) GetAllCallTrees() map[int]*FunctionCallNode {
	c.mu.RLock()
	defer c.mu.RUnlock()

	trees := make(map[int]*FunctionCallNode)
	for k, v := range c.callTrees {
		trees[k] = v
	}
	return trees
}

// GetNodeByID 通过CallID获取节点
func (c *CallStackCollector) GetNodeByID(callID uint64) *FunctionCallNode {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.nodePool[callID]
}

// 打印调用树（递归）
func (c *CallStackCollector) PrintCallTree(goroutineID int) {
	tree := c.GetCallTree(goroutineID)
	if tree == nil {
		fmt.Printf("Goroutine %d: 没有调用记录\n", goroutineID)
		return
	}

	fmt.Printf("=== Goroutine %d 调用树 ===\n", goroutineID)
	c.printNode(tree, 0)
}

func (c *CallStackCollector) printNode(node *FunctionCallNode, indent int) {
	if node == nil {
		return
	}

	// 计算缩进
	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}

	// 计算持续时间
	var duration int64
	duration = -1 //-1表示还在运行
	if node.EndUnixNano != 0 {
		duration = node.EndUnixNano - node.StartUnixNano
	}

	fmt.Printf("%s├─ [%d] %s\n", prefix, node.CallID, node.FuncName)
	fmt.Printf("%s│   Goroutine: %d, 深度: %d\n", prefix, node.GoroutineID, node.Depth)
	fmt.Printf("%s│   开始: %v\n", prefix, node.StartUnixNano)
	fmt.Printf("%s│   结束: %v\n", prefix, node.EndUnixNano)
	fmt.Printf("%s│   耗时: %v\n", prefix, duration)

	// 递归打印子节点
	for _, child := range node.Children {
		c.printNode(child, indent+1)
	}
}

// 获取扁平化的调用链（按时间顺序）
func (c *CallStackCollector) GetFlatCallChain(goroutineID int) []*FunctionCallNode {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []*FunctionCallNode
	tree := c.callTrees[goroutineID]
	if tree == nil {
		return result
	}

	// 深度优先遍历收集所有节点
	var dfs func(*FunctionCallNode)
	dfs = func(node *FunctionCallNode) {
		if node == nil {
			return
		}
		result = append(result, node)
		for _, child := range node.Children {
			dfs(child)
		}
	}

	dfs(tree)
	return result
}

// 获取当前活动调用栈（从当前函数回溯到根）
func (c *CallStackCollector) GetCurrentStack(goroutineID int) []*FunctionCallNode {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if stack, exists := c.callStackMap[goroutineID]; exists {
		// 返回副本
		result := make([]*FunctionCallNode, len(stack))
		copy(result, stack)
		return result
	}
	return nil
}

// 计算调用统计信息
type CallStatistics struct {
	TotalCalls int
	TotalTime  int64
	AvgTime    int64
	MaxTime    int64
	MinTime    int64
	ByFunction map[string]FunctionStats
}

type FunctionStats struct {
	CallCount int
	TotalTime int64
	AvgTime   int64
}

func (c *CallStackCollector) GetStatistics(goroutineID int) *CallStatistics {
	nodes := c.GetFlatCallChain(goroutineID)
	if len(nodes) == 0 {
		return nil
	}

	stats := &CallStatistics{
		TotalCalls: len(nodes),
		ByFunction: make(map[string]FunctionStats),
	}

	// 统计总体
	var totalTime int64
	var maxTime, minTime int64
	first := true

	for _, node := range nodes {
		if node.EndUnixNano != 0 {
			d := node.EndUnixNano - node.StartUnixNano
			totalTime += d

			if first {
				maxTime = d
				minTime = d
				first = false
			} else {
				if d > maxTime {
					maxTime = d
				}
				if d < minTime {
					minTime = d
				}
			}

			// 按函数统计
			funcStat := stats.ByFunction[node.FuncName]
			funcStat.CallCount++
			funcStat.TotalTime += d
			funcStat.AvgTime = funcStat.TotalTime / int64(funcStat.CallCount)
			stats.ByFunction[node.FuncName] = funcStat
		}
	}

	stats.TotalTime = totalTime
	if stats.TotalCalls > 0 {
		stats.AvgTime = totalTime / int64(stats.TotalCalls)
	}
	stats.MaxTime = maxTime
	stats.MinTime = minTime

	return stats
}
