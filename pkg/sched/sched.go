package sched

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
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
