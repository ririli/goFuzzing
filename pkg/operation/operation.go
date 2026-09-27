package operation

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"toolkit/pkg/function"
	"toolkit/pkg/goroutine"
)

var event sync.Map

type completion struct {
	once sync.Once
	done chan struct{}
}

func operationCompletion(id uint64) *completion {
	value, _ := event.LoadOrStore(id, &completion{done: make(chan struct{})})
	return value.(*completion)
}

func completeOperation(id uint64) {
	c := operationCompletion(id)
	c.once.Do(func() { close(c.done) })
}

var timeout, recovertimeout time.Duration

var config *Config
var cancel chan struct{}

var once sync.Once

var debugSched bool

func init() {
	config = NewConfig()
	cancel = make(chan struct{})
	timeout = time.Second * 5 // /5 = 1s OP 配对超时
	recovertimeout = time.Second * 1

	// SCHED_DEBUG=1 时关闭操作详情日志（fuzzing执行阶段）
	debugSched = os.Getenv("SCHED_DEBUG") != "1"
}

// 查找当前等待 ID 对应的前置操作
func (c *Config) findPrev(opId uint64) []uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if prevId, ok := c.preOpMap[opId]; ok {
		return prevId
	}
	return nil
}

func ParsePair(s string) {
	config.mu.Lock()
	defer config.mu.Unlock()

	for len(s) > 0 {
		left := strings.Index(s, "(")
		if left == -1 {
			break
		}
		right := strings.Index(s[left:], ")")
		if right == -1 {
			break
		}
		right += left

		pairStr := s[left+1 : right]
		ids := strings.Split(pairStr, ",")
		if len(ids) != 2 {
			s = s[right+1:]
			continue
		}

		preId, err1 := strconv.ParseUint(strings.TrimSpace(ids[0]), 10, 64)
		nextId, err2 := strconv.ParseUint(strings.TrimSpace(ids[1]), 10, 64)
		if err1 != nil || err2 != nil {
			s = s[right+1:]
			continue
		}

		config.active[preId] = struct{}{}
		config.active[nextId] = struct{}{}
		config.preOpMap[nextId] = append(config.preOpMap[nextId], preId)
		config.waitMap[nextId]++

		s = s[right+1:]
	}
}

func ParseInput() {
	input_pairs := os.Getenv("InputOp")
	if input_pairs != "" {
		ParsePair(input_pairs)
	}
}

func (c *Config) doWait(id uint64) (wait bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if _, ok := c.active[id]; !ok {
		return false
	}
	if v, ok := c.waitMap[id]; ok {
		if v >= 1 {
			return true
		}
	}
	return false
}

func (c *Config) waitDec(id uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if v, ok := c.waitMap[id]; ok {
		if v <= 1 {
			delete(c.waitMap, id)
		}
		if v > 1 {
			c.waitMap[id]--
		}
	}
}

// InstChBF channel 操作前拦截，根据调度配置决定是否等待前置 opId 完成
func InstChBF(opId uint64) {
	if !config.doWait(opId) {
		return
	}
	for _, preID := range config.findPrev(opId) {
		timer := time.NewTimer(timeout / 5)
		select {
		case <-operationCompletion(preID).done:
			config.waitDec(opId)
			fmt.Printf("{COVERED_OP} {%v, %v}\n", preID, opId)
		case <-cancel:
			timer.Stop()
			return
		case <-timer.C:
			fmt.Printf("{TIMEOUT_OP} {%v, %v}\n", preID, opId)
			return
		}
		timer.Stop()
	}
}

// formatFids 将当前函数栈格式化为 "3,7" 形式，无插桩函数在栈上时返回空串。
// function 粒度下 fuzzer 据此将操作归属到函数；goroutine 粒度下
// FunctionPass 未注入，栈恒为空，字段不输出，协议保持兼容。
func formatFids() string {
	fids := function.CurrentFuncStack()
	if len(fids) == 0 {
		return ""
	}
	parts := make([]string, len(fids))
	for i, fid := range fids {
		parts[i] = strconv.FormatUint(fid, 10)
	}
	return strings.Join(parts, ",")
}

// InstChAF channel 操作后记录，用于通知等待者并输出 ObjectID 日志
func InstChAF[T any | chan T | <-chan T | chan<- T](opId uint64, o T, opType string) {
	if debugSched {
		addr := uint64(reflect.ValueOf(o).Pointer())
		gid := goroutine.CurrentGid()
		print("[FB]chan: obj=", addr, "; opId=", opId, "; gid=", gid, "; op=", opType, ";", formatFidsField(), "\n")
	}
	completeOperation(opId)
}

// formatFidsField 返回 [FB] 日志中的 fids 字段（含分号），栈为空时返回空串。
func formatFidsField() string {
	if s := formatFids(); s != "" {
		return " fids=" + s + ";"
	}
	return ""
}

// InstChSelectAF select 中的 channel 操作后记录（仅 AF，无 BF）
// 与 InstChAF 的区别：输出 select=1 标记，fuzzer 据此限制该操作只能做 Op1（pre）
func InstChSelectAF[T any | chan T | <-chan T | chan<- T](opId uint64, o T, opType string) {
	InstChSelectAFAddr(opId, uint64(reflect.ValueOf(o).Pointer()), opType)
}

// CaptureSelectChannel preserves Go's select operand evaluation order.
func CaptureSelectChannel[T any](ch T, addr *uint64) T {
	*addr = uint64(reflect.ValueOf(ch).Pointer())
	return ch
}

func InstChSelectAFAddr(opId uint64, addr uint64, opType string) {
	if debugSched {
		gid := goroutine.CurrentGid()
		print("[FB]chan: obj=", addr, "; opId=", opId, "; gid=", gid, "; op=", opType, "; select=1;", formatFidsField(), "\n")
	}
	completeOperation(opId)
}

// InstWgBF WaitGroup 操作前拦截（Add/Done/Wait）
func InstWgBF(opId uint64) { InstChBF(opId) }

// InstWgAF WaitGroup 操作后记录
func InstWgAF(opId uint64, wg any, opType string) {
	if debugSched {
		addr := uint64(reflect.ValueOf(wg).Pointer())
		gid := goroutine.CurrentGid()
		print("[FB]wg: obj=", addr, "; opId=", opId, "; gid=", gid, "; op=", opType, ";", formatFidsField(), "\n")
	}
	completeOperation(opId)
}
