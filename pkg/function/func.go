// Package function 提供函数颗粒度 fuzzing 的运行时支持，对标 pkg/goroutine：
// PointControl 双栏断点控制（control.go）、Trace 生命周期追踪（本文件）、
// 并发对检测/推测与 stderr 输出（overlap.go/infer.go）、Input 解析（parse.go）。
package function

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

var (
	cfg     *Config      // 函数级调度配置
	tracker *FuncTracker // 函数调用生命周期追踪器

	skipRecord bool
)

func init() {
	cfg = NewConfig()
	tracker = NewFuncTracker()
	skipRecord = os.Getenv("RECORD_STACK") == "1"
}

// getGoroutineID 获取当前运行时goroutine ID（用于按 OS 协程维护调用栈）
func getGoroutineID() int {
	buf := make([]byte, 64)
	n := runtime.Stack(buf, false)
	var id int
	if n > 0 {
		fmt.Sscanf(string(buf[:n]), "goroutine %d", &id)
	}
	return id
}

// Trace 函数级别入口hook，在函数入口以 defer Trace(funcID)() 形式调用，
// 对标 goroutine.Enter/Exit：记录函数生命周期（参与并发重叠检测与调用边聚合）。
// 断点控制由插桩单独注入的 PointControl 完成（作为第一条语句，确保 rendezvous 尽早发生）。
func Trace(funcID uint64) func() {
	if skipRecord {
		return func() {}
	}
	gid := getGoroutineID()
	startTime := time.Now().UnixNano()

	tracker.mu.Lock()
	callID := tracker.nextCallID
	tracker.nextCallID++

	// 从调用栈确定父函数（caller）
	var parentID uint64
	stack := tracker.depth[gid]
	if len(stack) > 0 {
		parentID = stack[len(stack)-1].FuncID
	}
	tracker.callerMap[funcID] = parentID

	rec := &FuncRecord{
		FuncID:    funcID,
		CallID:    callID,
		ParentID:  parentID,
		StartTime: startTime,
	}
	tracker.funcMap[funcID] = append(tracker.funcMap[funcID], rec)
	tracker.depth[gid] = append(tracker.depth[gid], rec)
	tracker.mu.Unlock()

	return func() {
		if skipRecord {
			return
		}
		tracker.mu.Lock()
		rec.EndTime = time.Now().UnixNano()
		// 出栈
		s := tracker.depth[gid]
		for i := len(s) - 1; i >= 0; i-- {
			if s[i].CallID == callID {
				tracker.depth[gid] = s[:i]
				break
			}
		}
		tracker.mu.Unlock()
	}
}

// CurrentFuncStack 返回当前 OS goroutine 调用栈上所有未退出的被插桩函数 ID。
// 供 operation 包在 [FB] 日志中归属操作所属函数：一个操作归属于栈上全部函数，
// 对应 goroutine 粒度下"操作归属于所在协程"的语义。
// funcID=0（主测试函数 EnterMain 压栈）不参与归属。
func CurrentFuncStack() []uint64 {
	gid := getGoroutineID()
	tracker.mu.RLock()
	defer tracker.mu.RUnlock()

	stack := tracker.depth[gid]
	fids := make([]uint64, 0, len(stack))
	for _, rec := range stack {
		if rec.FuncID != 0 {
			fids = append(fids, rec.FuncID)
		}
	}
	return fids
}

// EnterMain 记录主测试函数入口（funcID=0），对标 goroutine.EnterMain。
// 在生成的 TestXxx_1 包装函数开头调用；funcID=0 不参与断点控制。
func EnterMain() {
	if skipRecord {
		return
	}
	gid := getGoroutineID()
	tracker.mu.Lock()
	rec := &FuncRecord{
		FuncID:    0,
		CallID:    tracker.nextCallID,
		ParentID:  0,
		StartTime: time.Now().UnixNano(),
	}
	tracker.nextCallID++
	tracker.funcMap[0] = append(tracker.funcMap[0], rec)
	tracker.depth[gid] = append(tracker.depth[gid], rec)
	tracker.mu.Unlock()
}

// ExitMain 记录主测试函数出口。
// 在生成的 TestXxx_1 包装函数中defer调用
func ExitMain() {
	if skipRecord {
		return
	}
	gid := getGoroutineID()
	tracker.mu.Lock()
	if instances := tracker.funcMap[0]; len(instances) > 0 {
		instances[len(instances)-1].EndTime = time.Now().UnixNano()
	}
	tracker.depth[gid] = nil
	tracker.mu.Unlock()
}
