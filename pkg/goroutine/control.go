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
	arrived  int32         // 原子计数：0 → 1 → 2
	release  chan struct{} // 计数到 2 时关闭
	expired  int32         // 原子标志：0=活跃，1=已超时
}

// pointControl 双栏断点控制。
//
// 对于 gid 参与的每一个 barrier：
//   - 原子递增 arrived 计数
//   - 如果计数达到 2（第二个到达）→ 关闭 release channel，打印 {COVERED}
//   - 如果计数为 1（第一个到达）→ 阻塞等待 release 或超时
//   - 超时时设置 expired 标志，后续到达者不再操作
func (c *Config) pointControl(gid uint64) {
	if atomic.LoadUint32(&c.hasActive) == 0 {
		return
	}
	if !c.isActive(gid) {
		return
	}

	c.mu.RLock()
	gates := c.barriers[gid]
	c.mu.RUnlock()

	for _, gate := range gates {
		count := atomic.AddInt32(&gate.arrived, 1)

		if count == 2 {
			// 第二个到达 → 释放栅栏
			if atomic.LoadInt32(&gate.expired) == 0 {
				close(gate.release)
				fmt.Printf("{COVERED} {%v, %v}\n", gate.id1, gate.id2)
			}
		} else {
			// 第一个到达 → 等待伙伴
			select {
			case <-gate.release:
				// 被第二个到达者释放，已打印 {COVERED}
			case <-time.After(c.BarrierTimeout):
				atomic.StoreInt32(&gate.expired, 1)
				fmt.Printf("{TIMEOUT} {%v, %v}\n", gate.id1, gate.id2)
			}
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
