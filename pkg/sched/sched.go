package sched

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"
	"toolkit/pkg/sched/goleak"
)

var event sync.Map
var timeout, recovertimeout time.Duration

var config *Config
var cancel chan struct{}

var once sync.Once

const (
	debugSched = true
)

func init() {
	config = NewConfig()
	cancel = make(chan struct{})
	timeout = time.Second * 20
	recovertimeout = time.Second * 1
	if s := os.Getenv("TIMEOUT"); s != "" {
		t, err := strconv.ParseInt(s, 10, 32)
		if err == nil {
			timeout = time.Duration(t) * time.Second
		}
	}
	if s := os.Getenv("RECOVER_TIMEOUT"); s != "" {
		t, err := strconv.ParseInt(s, 10, 32)
		if err == nil {
			recovertimeout = time.Duration(t) * time.Millisecond
		}
	}
}

// find sender with current wait ID
func (c *Config) findPrev(opId uint64) []uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if prevId, ok := c.preOpMap[opId]; ok {
		return prevId
	}
	return nil
}

// 1. add the pairs to wait_queue
// 2. add to the active
// 3. add the next IDs to waitmap with a counter
func ParsePair(s string) {

}

func ParseInput() {
	input_pairs := os.Getenv("Input_op")
	if input_pairs != "" {
		ParsePair(input_pairs)
	}
}

func SetTimeout(s int) {
	timeout = time.Second * time.Duration(s)
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
func InstChBF[T any | chan T | <-chan T | chan<- T](opId uint64, o T, funcId uint64, opType string) {

	if !config.doWait(opId) {
		return
	}
	preIds := config.findPrev(opId)
	if len(preIds) == 0 {
		return
	}
	for _, preId := range preIds {
		timer := time.After(timeout / 5)
		for {
			if _, ok := event.LoadAndDelete(preId); ok {
				config.waitDec(opId)
				fmt.Printf("[COVERED] {%v, %v}\n", preId, opId)
				break
			}
			select {
			case <-cancel:
				return
			case <-timer:
				return
			default:
			}
		}
	}
}

// InstChAF channel 操作后记录，用于通知等待者并输出 ObjectID 日志
func InstChAF[T any | chan T | <-chan T | chan<- T](opId uint64, o T, funcId uint64, opType string) {
	if debugSched {
		addr := uint64(reflect.ValueOf(o).Pointer())
		print("[FB]chan: obj=", addr, "; opId=", opId, "; funcId=", funcId, "; op=", opType, ";\n")
	}
	event.Store(opId, struct{}{})
}

// InstWgBF WaitGroup 操作前拦截（Add/Done/Wait）
func InstWgBF(opId uint64, wg *sync.WaitGroup, funcId uint64, opType string) {
	if !config.doWait(opId) {
		return
	}
	preIds := config.findPrev(opId)
	if len(preIds) == 0 {
		return
	}
	for _, preId := range preIds {
		timer := time.After(timeout / 5)
		for {
			if _, ok := event.LoadAndDelete(preId); ok {
				config.waitDec(opId)
				fmt.Printf("[COVERED] {%v, %v}\n", preId, opId)
				break
			}
			select {
			case <-cancel:
				return
			case <-timer:
				return
			default:
			}
		}
	}
}

// InstWgAF WaitGroup 操作后记录
func InstWgAF(opId uint64, wg *sync.WaitGroup, funcId uint64, opType string) {
	if debugSched {
		addr := uint64(reflect.ValueOf(wg).Pointer())
		print("[FB]wg: obj=", addr, "; opId=", opId, "; funcId=", funcId, "; op=", opType, ";\n")
	}
	event.Store(opId, struct{}{})
}

func GetDone() chan struct{} {
	return make(chan struct{})
}

func GetTimeout() <-chan time.Time {
	return time.After(timeout)
}

func Done(ch chan struct{}) {
	close(ch)
}

func baseCheck(t *testing.T) {
	opts := []goleak.Option{
		goleak.IgnoreTopFunction("time.Sleep"),
		goleak.IgnoreTopFunction("testing.(*F).Fuzz.func1"),
		goleak.IgnoreTopFunction("testing.runFuzzTests"),
		goleak.IgnoreTopFunction("testing.runFuzzing"),
		goleak.IgnoreTopFunction("os/signal.NotifyContext.func1"),
		goleak.IgnoreTopFunction("testing.tRunner.func1"),
		goleak.IgnoreTopFunction("github.com/ethereum/go-ethereum/metrics.(*meterArbiter).tick"),
		goleak.IgnoreTopFunction("github.com/ethereum/go-ethereum/core.(*txSenderCacher).cache"),
		goleak.IgnoreTopFunction("github.com/ethereum/go-ethereum/consensus/ethash.(*remoteSealer).loop"),
		goleak.MaxRetryAttempts(24),
		goleak.MaxSleepInterval(10 * time.Second),
	}

	goleak.VerifyNone(t, opts...)
}
