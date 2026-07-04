package goroutine

import (
	"fmt"
	"sync/atomic"
	"time"
)

// pointControl 实现goroutine之间的断点控制
func pointControl(gid uint64) {
	if atomic.LoadUint32(&cfg.hasActive) == 0 {
		return
	}
	if !cfg.isActive(gid) {
		return
	}
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
	if val, ok := waiters.Load(id); ok {
		return val.(chan struct{})
	}

	newWaiter := make(chan struct{}, 1)
	actual, _ := waiters.LoadOrStore(id, newWaiter)
	return actual.(chan struct{})
}

// completeOperation 标记goroutine完成，通知所有等待者
func completeOperation(id uint64) {
	if val, ok := waiters.LoadAndDelete(id); ok {
		ch := val.(chan struct{})
		select {
		case <-ch:
		default:
			close(ch)
		}
		return
	}
	done := make(chan struct{})
	close(done)
	waiters.LoadOrStore(id, done)
}
