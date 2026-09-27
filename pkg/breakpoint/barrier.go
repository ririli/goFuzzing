package breakpoint

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// BarrierConfig 双栏等待策略。
//
// 与单栏 Config 不同，双栏不区分"谁等谁"——对于一个函数对 (a,b)，
// 双方都必须到达入口点后才同时放行。这创造了真正的"同时开始并发"场景。
//
// 信号输出格式与单栏完全一致，因此无需修改 fuzzer 即可切换使用。
type BarrierConfig struct {
	mu         sync.RWMutex
	barriers   map[uint64][]*barrierGate // funcID → 该函数参与的所有 barrier
	activeFunc map[uint64]struct{}       // 需要参与 barrier 的函数集合
	hasActive  uint32                    // 原子标志，是否有活跃配置
	Timeout    time.Duration             // 单个 barrier 等待超时
}

// barrierGate 表示一对函数的双栏栅栏。
// 两个函数都到达时 release channel 被关闭，同时放行。
type barrierGate struct {
	id1, id2 uint64        // 该 barrier 对应的两个函数 ID
	arrived  int32         // 位图：1/2=各侧到达，3=覆盖，4=超时
	release  chan struct{} // 计数到 2 时关闭
	timedOut int32         // 原子标志：0=活跃，1=已超时
}

// NewBarrierConfig 创建双栏配置
func NewBarrierConfig() *BarrierConfig {
	return &BarrierConfig{
		barriers:   make(map[uint64][]*barrierGate),
		activeFunc: make(map[uint64]struct{}),
		Timeout:    40 * time.Millisecond,
	}
}

// HasActive 返回是否有活跃的双栏配置
func (b *BarrierConfig) HasActive() bool {
	return atomic.LoadUint32(&b.hasActive) == 1
}

// IsActive 判断函数是否参与双栏同步
func (b *BarrierConfig) IsActive(funcId uint64) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, ok := b.activeFunc[funcId]
	return ok
}

// ParseInput 从环境变量 Input 解析可疑函数对
func (b *BarrierConfig) ParseInput() {
	inputSusPairs := os.Getenv("Input")
	if inputSusPairs != "" {
		b.ParseSusPairs(inputSusPairs)
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.activeFunc) > 0 {
		atomic.StoreUint32(&b.hasActive, 1)
	}
}

// ParseSusPairs 解析输入的函数对。
// 格式与单栏相同: (id1,id2)(id3,id4)...
// 语义不同: 每一对的两个函数互为等待对象，双方都到达后才放行。
func (b *BarrierConfig) ParseSusPairs(s string) {
	b.mu.Lock()
	defer b.mu.Unlock()

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

		var id1, id2 uint64
		_, err := fmt.Sscanf(pairStr, "%d,%d", &id1, &id2)
		if err == nil && id1 != 0 && id2 != 0 && id1 != id2 {
			b.activeFunc[id1] = struct{}{}
			b.activeFunc[id2] = struct{}{}

			gate := &barrierGate{
				id1:     id1,
				id2:     id2,
				release: make(chan struct{}),
			}
			// 双方共享同一个 gate
			b.barriers[id1] = append(b.barriers[id1], gate)
			b.barriers[id2] = append(b.barriers[id2], gate)
		}

		s = s[right+1:]
	}
}

// PointControl requires one arrival from each distinct side, once per execution.
func (b *BarrierConfig) PointControl(funcId uint64) {
	if !b.HasActive() {
		return
	}
	if !b.IsActive(funcId) {
		return
	}

	b.mu.RLock()
	gates := append([]*barrierGate(nil), b.barriers[funcId]...)
	b.mu.RUnlock()

	for _, gate := range gates {
		bit := int32(1)
		if funcId == gate.id2 {
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
			timer := time.NewTimer(b.Timeout)
			select {
			case <-gate.release:
			case <-timer.C:
				if atomic.CompareAndSwapInt32(&gate.arrived, bit, 4) {
					atomic.StoreInt32(&gate.timedOut, 1)
					close(gate.release)
					fmt.Printf("{TIMEOUT} {%v, %v}\n", gate.id1, gate.id2)
				}
			}
			timer.Stop()
			break
		}
	}
}
