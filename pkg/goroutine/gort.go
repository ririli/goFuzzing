package goroutine

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	tracker *GoroutineTracker // goroutine生命周期追踪器
	mu      sync.Mutex

	waiters sync.Map
	gidMap  sync.Map // OS goroutine ID(int) → gid(uint64)
)

var (
	cfg           *Config
	timeout       time.Duration
	timeoutGlobal time.Duration
)

func init() {
	cfg = NewConfig()
	timeout = 40 * time.Millisecond
	tracker = NewGoroutineTracker()
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

// ParseInput 解析环境变量Input中的调度配置
func ParseInput() {
	input_susPairs := os.Getenv("Input")
	if input_susPairs != "" {
		ParsePairs(input_susPairs)
	}
	if len(cfg.activeMap) > 0 {
		atomic.StoreUint32(&cfg.hasActive, 1)
	}
}

// ParsePairs 解析输入的goroutine对
// 格式: (id1,id2)(id3,id4)(id5,id6)...
func ParsePairs(s string) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

	// 逐对解析 (id1,id2) 格式
	for len(s) > 0 {
		// 查找左括号
		left := strings.Index(s, "(")
		if left == -1 {
			break
		}
		// 查找右括号
		right := strings.Index(s[left:], ")")
		if right == -1 {
			break
		}
		right += left // 调整为绝对位置

		// 提取括号内的内容
		pairStr := s[left+1 : right]

		// 解析两个 ID
		var id1, id2 uint64
		_, err := fmt.Sscanf(pairStr, "%d,%d", &id1, &id2)
		if err == nil {
			cfg.activeMap[id1] = struct{}{}
			cfg.activeMap[id2] = struct{}{}
			cfg.preMap[id2] = append(cfg.preMap[id2], id1)
			actual, _ := cfg.waitMap.LoadOrStore(id2, new(atomic.Int32))
			actual.(*atomic.Int32).Add(1)
		}

		// 移动到下一对
		s = s[right+1:]
	}
}

// EnterMain 记录主goroutine (gid=0) 的开始时间
// 在生成的 TestXxx_1 包装函数开头调用
func EnterMain() {
	gidMap.Store(getCurrentGoroutineID(), uint64(0))
	tracker.EnterGoroutineWithParent(0, 0) // 主goroutine没有父goroutine
}

// ExitMain 记录主goroutine的结束时间并通知等待者
// 在生成的 TestXxx_1 包装函数中defer调用
func ExitMain() {
	gidMap.Delete(getCurrentGoroutineID())
	tracker.ExitGoroutine(0)
	completeOperation(0)
}

// Enter goroutine级别入口hook，在go语句创建的goroutine开始时调用
func Enter(gid uint64) {
	// 查找父goroutine ID（创建当前goroutine的那个goroutine）
	parentGid := CurrentGid()
	// 存储当前OS goroutine → gid的映射
	gidMap.Store(getCurrentGoroutineID(), gid)

	pointControl(gid)
	tracker.EnterGoroutineWithParent(gid, parentGid)
}

// Exit goroutine级别出口hook，在go语句创建的goroutine结束时defer调用
func Exit(gid uint64) {
	gidMap.Delete(getCurrentGoroutineID())
	tracker.ExitGoroutine(gid)
	completeOperation(gid)
}

// pointControl 实现goroutine之间的断点控制
func pointControl(gid uint64) {
	if atomic.LoadUint32(&cfg.hasActive) == 0 {
		return
	}
	if !cfg.isActive(gid) {
		return
	}
	// 判断goroutine是否需要等待
	if cfg.doWait(gid) {
		preIds := cfg.findPrev(gid)
		if preIds != nil {
			for _, preId := range preIds {
				waiter := getWaiter(preId)

				select {
				case <-waiter:
					cfg.waitMapDec(gid)
					fmt.Printf("{COVERED} {%v, %v}\n", preId, gid)
				case <-time.After(timeout):
					fmt.Printf("{TIMEOUT} {%v, %v}\n", preId, gid)
				}
			}
		}
	}
	completeOperation(gid)
}

// isActive 判断goroutine是否处于活动调度状态
func (c *Config) isActive(gid uint64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.activeMap[gid]
	return ok
}

// findPrev 查找指定goroutineId的前驱goroutine ID列表
func (c *Config) findPrev(gid uint64) []uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if prevId, ok := c.preMap[gid]; ok {
		return prevId
	}
	return nil
}

// doWait 判断goroutine是否需要等待
func (c *Config) doWait(gid uint64) bool {
	if value, ok := c.waitMap.Load(gid); ok {
		return value.(*atomic.Int32).Load() > 0
	}
	return false
}

// waitMapDec 减少指定goroutine ID的等待计数
func (c *Config) waitMapDec(gid uint64) {
	if val, ok := c.waitMap.Load(gid); ok {
		newVal := val.(*atomic.Int32).Add(-1)
		if newVal <= 0 {
			c.waitMap.Delete(gid)
		}
	}
}

// getWaiter 获取或创建指定goroutine ID的等待channel
func getWaiter(id uint64) chan struct{} {
	// 尝试加载已存在的 waiter
	if val, ok := waiters.Load(id); ok {
		return val.(chan struct{})
	}

	// 创建新的 waiter
	newWaiter := make(chan struct{}, 1) // 缓冲为 1，避免发送时阻塞

	// 存储，如果已被其他协程创建则使用已有的
	actual, _ := waiters.LoadOrStore(id, newWaiter)
	return actual.(chan struct{})
}

// completeOperation 标记goroutine完成，通知所有等待者
func completeOperation(id uint64) {
	if val, ok := waiters.LoadAndDelete(id); ok {
		ch := val.(chan struct{})
		// 安全关闭：使用 select 检测 channel 是否已被关闭
		// LoadAndDelete 保证原子删除，但 channel 可能已被 else 分支预先关闭
		select {
		case <-ch:
			// channel 已关闭，无需重复关闭
		default:
			close(ch)
		}
		return
	}
	// 向前引用：completeOperation 发生在任何 getWaiter 之前
	// 创建一个预先关闭的 channel，后续 getWaiter 会直接返回
	done := make(chan struct{})
	close(done)
	waiters.LoadOrStore(id, done)
}

// PrintGoroutinePairs 打印所有goroutine并发对到stderr
// 包含直接观测（Rule 0）和推测（Rule 2: 父子, Rule 3: 自配对）
// 格式：[COVERED] 或 [SUSPECT] gid1,gid2|file1:line1,file2:line2|confidence|sourceType;
func PrintGoroutinePairs() {
	time.Sleep(500 * time.Millisecond) // 等待子goroutine执行完毕

	// Rule 0: 直接观测的时间重叠对 → [COVERED]
	observedPairs := tracker.DetectGoroutineOverlaps()

	// Rule 2: 父子goroutine对 → [SUSPECT]
	parentChildPairs := tracker.InferParentChildPairs()

	// Rule 3: 自配对（同gid多实例） → [SUSPECT]
	selfPairs := tracker.InferSelfPairs()

	total := len(observedPairs) + len(parentChildPairs) + len(selfPairs)
	if total == 0 {
		print("No concurrent goroutine pairs found.\n")
		return
	}

	// 输出直接观测对 [COVERED]
	for _, pair := range observedPairs {
		print(pair.String())
	}

	// 输出推测对 [SUSPECT]
	for _, pair := range parentChildPairs {
		print(pair.String())
	}
	for _, pair := range selfPairs {
		print(pair.String())
	}
}
