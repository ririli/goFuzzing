package goroutine

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
)

var (
	tracker *GoroutineTracker // goroutine生命周期追踪器
	mu      sync.Mutex

	gidMap sync.Map // OS goroutine ID(int) → gid(uint64)
)

var (
	cfg *Config
)

var skipRecord bool

func init() {
	cfg = NewConfig()
	tracker = NewGoroutineTracker()
	skipRecord = os.Getenv("RECORD_STACK") == "1"
}

// CurrentGid 返回当前OS goroutine对应的静态goroutine ID
// 供sched包在[FB]日志中使用
func CurrentGid() uint64 {
	id := getCurrentGoroutineID()
	if val, ok := gidMap.Load(id); ok {
		return val.(uint64)
	}
	return 0
}

// getCurrentGoroutineID 获取当前运行时goroutine ID
func getCurrentGoroutineID() int {
	buf := make([]byte, 64)
	n := runtime.Stack(buf, false)
	var id int
	if n > 0 {
		fmt.Sscanf(string(buf[:n]), "goroutine %d", &id)
	}
	return id
}

// EnterMain 记录主goroutine (gid=0) 的开始时间
// 在生成的 TestXxx_1 包装函数开头调用
func EnterMain() {
	if skipRecord {
		return
	}
	gidMap.Store(getCurrentGoroutineID(), uint64(0))
	tracker.EnterGoroutineWithParent(0, 0) // 主goroutine没有父goroutine
}

// ExitMain 记录主goroutine的结束时间
// gid=0 不参与断点控制
// 在生成的 TestXxx_1 包装函数中defer调用
func ExitMain() {
	if skipRecord {
		return
	}
	gidMap.Delete(getCurrentGoroutineID())
	tracker.ExitGoroutine(0)
}

// Enter goroutine级别入口hook，在go语句创建的goroutine开始时调用
// parentGid 由调用方在父goroutine上下文中通过 goroutine.CurrentGid() 获取并传入
// pointControl 作为第一条语句，确保 rendezvous 尽早发生
func Enter(gid uint64, parentGid uint64) {
	cfg.pointControl(gid)

	if skipRecord {
		return
	}
	// 存储当前OS goroutine → gid的映射
	gidMap.Store(getCurrentGoroutineID(), gid)
	tracker.EnterGoroutineWithParent(gid, parentGid)
}

// Exit goroutine级别出口hook，在go语句创建的goroutine结束时defer调用
// 注意：断点控制在 Enter→pointControl 入口处完成
func Exit(gid uint64) {
	if skipRecord {
		return
	}
	gidMap.Delete(getCurrentGoroutineID())
	tracker.ExitGoroutine(gid)
}

// PrintRecords 打印所有goroutine实例的追踪记录（含父子关系）
// 用于调试时查看完整的goroutine生命周期数据
func PrintRecords() {
	if skipRecord {
		return
	}
	time.Sleep(500 * time.Millisecond) // 等待子goroutine执行完毕
	tracker.PrintGoroutineRecords()
}
