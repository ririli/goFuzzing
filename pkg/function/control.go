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
	arrived  int32         // 位图：1/2=各侧到达，3=覆盖，4=超时
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
	gates := append([]*barrierGate(nil), cfg.barriers[funcID]...)
	cfg.mu.RUnlock()

	for _, gate := range gates {
		bit := int32(1)
		if funcID == gate.id2 {
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
			timer := time.NewTimer(cfg.BarrierTimeout)
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
