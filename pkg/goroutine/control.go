package goroutine

import (
	"fmt"
	"sync/atomic"
	"time"
)

// barrierGate 表示一对goroutine的双栏栅栏。
// 两个goroutine都到达时 release channel 被关闭，同时放行。
type barrierGate struct {
	id1, id2 uint64        // 该 barrier 对应的两个 goroutine ID
	arrived  int32         // 位图：1/2=各侧到达，3=覆盖，4=超时
	release  chan struct{} // 计数到 2 时关闭
	expired  int32         // 原子标志：0=活跃，1=已超时
}

// pointControl requires one arrival per distinct side and a single terminal result.
func (c *Config) pointControl(gid uint64) {
	if atomic.LoadUint32(&c.hasActive) == 0 {
		return
	}
	if !c.isActive(gid) {
		return
	}

	c.mu.RLock()
	gates := append([]*barrierGate(nil), c.barriers[gid]...)
	c.mu.RUnlock()

	for _, gate := range gates {
		bit := int32(1)
		if gid == gate.id2 {
			bit = 2
		}
		for {
			state := atomic.LoadInt32(&gate.arrived)
			// Each side participates once; a completed gate never delays loops.
			if state >= 3 || state&bit != 0 {
				break
			}
			next := state | bit
			if !atomic.CompareAndSwapInt32(&gate.arrived, state, next) {
				continue
			}
			if next == 3 {
				close(gate.release)
				fmt.Printf("{COVERED} {%v, %v}\n", gate.id1, gate.id2)
				break
			}
			timer := time.NewTimer(c.BarrierTimeout)
			select {
			case <-gate.release:
			case <-timer.C:
				if atomic.CompareAndSwapInt32(&gate.arrived, bit, 4) {
					atomic.StoreInt32(&gate.expired, 1)
					close(gate.release)
					fmt.Printf("{TIMEOUT} {%v, %v}\n", gate.id1, gate.id2)
				}
			}
			timer.Stop()
			break
		}
	}
}

// isActive 判断goroutine是否处于活动调度状态
func (c *Config) isActive(gid uint64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.activeMap[gid]
	return ok
}

// HasActive 返回是否有活跃的断点配置
func (c *Config) HasActive() bool {
	return atomic.LoadUint32(&c.hasActive) == 1
}
