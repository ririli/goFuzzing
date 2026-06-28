package calltree

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

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

// ============================================================================
// 内部辅助函数
// ============================================================================

// getCurrentGoroutineID 获取当前goroutine ID（性能优化版本）
func getCurrentGoroutineID() int {
	buf := make([]byte, 64)
	n := runtime.Stack(buf, false)
	var id int
	if n > 0 {
		fmt.Sscanf(string(buf[:n]), "goroutine %d", &id)
	}
	return id
}

// getCallInfo 记录被插桩函数的name以及其被调用位置信息
func getCallInfo() (funcName string, callLoc CallLocation) {
	// level=3: runtime.Caller -> getCallInfo -> EnterFunction -> callstack.Trace -> defer闭包 -> 目标函数
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

	fn := runtime.FuncForPC(pc)
	if fn != nil {
		funcName = fn.Name()
	} else {
		funcName = "unknown"
	}

	callerPC, callerFile, callerLine, ok := runtime.Caller(level + 1)

	var finalFile string
	var finalLine int
	var tag string

	if ok {
		callerFn := runtime.FuncForPC(callerPC)
		if callerFn != nil {
			if isStandardLibrary(callerFile) {
				finalFile = selfFile
				finalLine = selfLine
				if strings.Contains(callerFile, "src/testing") {
					tag = " (Test)"
				} else if strings.Contains(callerFile, "src/runtime") {
					tag = " (go)"
				}
				finalFile += tag
			} else {
				finalFile = callerFile
				finalLine = callerLine
			}
		}
		relativeFile := convertToRelativePath(finalFile)

		callLoc = CallLocation{
			File:     relativeFile,
			Line:     finalLine,
			FuncName: callerFn.Name(),
			PC:       pc,
		}
		return funcName, callLoc
	}
	return "unknown", CallLocation{
		File:     "unknown",
		Line:     0,
		FuncName: "unknown",
		PC:       0,
	}
}

// convertToRelativePath 将绝对路径转换为相对路径
func convertToRelativePath(absPath string) string {
	keyDir := "gopie"
	idx := strings.Index(absPath, keyDir)
	if idx == -1 {
		return absPath
	}
	relativePath := absPath[idx:]
	relativePath = filepath.ToSlash(relativePath)
	return relativePath
}

// isStandardLibrary 判断是否是标准库或 runtime 的代码
func isStandardLibrary(filePath string) bool {
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

// cleanupStack 清理不一致的调用栈
func (c *CallStackCollector) cleanupStack(goroutineID int, callID uint64) {
	stack := c.callStackMap[goroutineID]
	for i := len(stack) - 1; i >= 0; i-- {
		if stack[i].CallID == callID {
			c.callStackMap[goroutineID] = stack[:i]
			return
		}
	}
}

// ============================================================================
// 公开方法
// ============================================================================

// EnterFunction 记录函数进入
func (c *CallStackCollector) EnterFunction(funcID uint64) *FunctionCallNode {
	goroutineID := getCurrentGoroutineID()
	funcName, callLoc := getCallInfo()
	c.mu.Lock()
	defer c.mu.Unlock()

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
	c.nodePool[node.CallID] = node
	c.funcIndex[funcID] = append(c.funcIndex[funcID], node.CallID)

	stack := c.callStackMap[goroutineID]
	if len(stack) == 0 {
		c.callTrees[goroutineID] = node
	} else {
		parent := stack[len(stack)-1]
		node.Parent = parent
		node.Depth = parent.Depth + 1
		parent.Children = append(parent.Children, node)
	}
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

	if stack, exists := c.callStackMap[goroutineID]; exists && len(stack) > 0 {
		if stack[len(stack)-1].CallID == node.CallID {
			c.callStackMap[goroutineID] = stack[:len(stack)-1]
		} else {
			c.cleanupStack(goroutineID, node.CallID)
		}
	}
}

// ClearActiveStacks 清理所有活动调用栈(收集结束后释放内存)
func (c *CallStackCollector) ClearActiveStacks() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.callStackMap = make(map[int][]*FunctionCallNode)
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

// GetFlatCallChain 获取扁平化的调用链（按时间顺序，深度优先）
func (c *CallStackCollector) GetFlatCallChain(goroutineID int) []*FunctionCallNode {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []*FunctionCallNode
	tree := c.callTrees[goroutineID]
	if tree == nil {
		return result
	}

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

// GetCurrentStack 获取当前活动调用栈（从当前函数回溯到根）
func (c *CallStackCollector) GetCurrentStack(goroutineID int) []*FunctionCallNode {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if stack, exists := c.callStackMap[goroutineID]; exists {
		result := make([]*FunctionCallNode, len(stack))
		copy(result, stack)
		return result
	}
	return nil
}

// ============================================================================
// 统计
// ============================================================================

// CallStatistics 调用统计信息
type CallStatistics struct {
	TotalCalls int
	TotalTime  int64
	AvgTime    int64
	MaxTime    int64
	MinTime    int64
	ByFunction map[string]FunctionStats
}

// FunctionStats 单个函数的统计信息
type FunctionStats struct {
	CallCount int
	TotalTime int64
	AvgTime   int64
}

// GetStatistics 获取指定goroutine的调用统计
func (c *CallStackCollector) GetStatistics(goroutineID int) *CallStatistics {
	nodes := c.GetFlatCallChain(goroutineID)
	if len(nodes) == 0 {
		return nil
	}

	stats := &CallStatistics{
		TotalCalls: len(nodes),
		ByFunction: make(map[string]FunctionStats),
	}

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

// ============================================================================
// 打印
// ============================================================================

// PrintCallTree 打印指定goroutine的调用树
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

	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}

	var duration int64
	duration = -1 // -1表示还在运行
	if node.EndUnixNano != 0 {
		duration = node.EndUnixNano - node.StartUnixNano
	}

	fmt.Printf("%s├─ [%d] %s\n", prefix, node.CallID, node.FuncName)
	fmt.Printf("%s│   Goroutine: %d, 深度: %d\n", prefix, node.GoroutineID, node.Depth)
	fmt.Printf("%s│   开始: %v\n", prefix, node.StartUnixNano)
	fmt.Printf("%s│   结束: %v\n", prefix, node.EndUnixNano)
	fmt.Printf("%s│   耗时: %v\n", prefix, duration)

	for _, child := range node.Children {
		c.printNode(child, indent+1)
	}
}
