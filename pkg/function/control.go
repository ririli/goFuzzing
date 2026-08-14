package function

import (
	"fmt"
	"sync/atomic"
	"time"
)

// barrierGate 表示一对函数 ID 之间的双栏栅栏。
// 两个函数都到达入口点才能放行。
type barrierGate struct {
	id1, id2 uint64        // 该 barrier 对应的两个函数 ID
	arrived  int32         // 原子计数：0 → 1 → 2
	release  chan struct{} // 计数到 2 时关闭
	expired  int32         // 原子标志：0=活跃，1=已超时
}

// PointControl 函数级双栏断点控制，对标 goroutine.Config.pointControl。
// 由插桩代码注入函数入口（第一条语句）；funcID 本轮未被调度时立即返回，
// 否则阻塞等待配对函数也到达其 PointControl，或等待 BarrierTimeout 超时。
func PointControl(funcID uint64) {
	if atomic.LoadUint32(&cfg.hasActive) == 0 {
		return
	}
	if !isActive(funcID) {
		return
	}

	cfg.mu.RLock()
	gates := cfg.barriers[funcID]
	cfg.mu.RUnlock()

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
			case <-time.After(cfg.BarrierTimeout):
				atomic.StoreInt32(&gate.expired, 1)
				fmt.Printf("{TIMEOUT} {%v, %v}\n", gate.id1, gate.id2)
			}
		}
	}
}

// isActive 判断函数是否处于活动调度状态
func isActive(funcID uint64) bool {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()
	_, ok := cfg.activeMap[funcID]
	return ok
}

// HasActive 返回是否有活跃的断点配置
func HasActive() bool {
	return atomic.LoadUint32(&cfg.hasActive) == 1
}
