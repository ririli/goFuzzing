// Package breakpoint 提供函数级别的双栏断点控制，用于函数颗粒度 fuzzing。
// 对标 pkg/goroutine，但操作对象是函数 ID 而非 goroutine ID。
// PointControl 由插桩代码注入调用；ParseInput 读取 Input 环境变量。
package breakpoint

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// barrierGate 表示一对函数 ID 之间的双栏栅栏。
// 两个函数都必须到达入口点才能放行。
type barrierGate struct {
	id1, id2 uint64
	arrived  int32         // 原子计数：0 → 1 → 2
	release  chan struct{} // 计数到 2 时关闭
	expired  int32         // 原子标志：0=活跃，1=已超时
}

// Config 保存函数级调度状态。
type Config struct {
	mu             sync.RWMutex
	barriers       map[uint64][]*barrierGate // funcID → 该 ID 参与的所有 barrierGate
	active         map[uint64]struct{}       // 本轮执行中需要调度的函数 ID 集合
	hasActive      uint32                    // 原子标志：1 表示 active 集合非空
	BarrierTimeout time.Duration
}

var cfg = &Config{
	barriers:       make(map[uint64][]*barrierGate),
	active:         make(map[uint64]struct{}),
	BarrierTimeout: 10 * time.Millisecond,
}

// PointControl 是注入到函数入口的双栏会合点。
// 如果 funcID 本轮未被调度，立即返回。
// 否则阻塞等待配对的函数也到达其 PointControl，
// 或等待 BarrierTimeout 超时。
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

// isActive 判断 funcID 是否在本轮调度中。
func isActive(funcID uint64) bool {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()
	_, ok := cfg.active[funcID]
	return ok
}

// HasActive 返回是否有活跃的断点配置。
func HasActive() bool {
	return atomic.LoadUint32(&cfg.hasActive) == 1
}

// ParseInput 读取 Input 环境变量，按 (id1,id2)... 格式
// 解析函数 ID 对并为每个对设置 barrierGate。
func ParseInput() {
	input := os.Getenv("Input")
	if input == "" {
		return
	}
	parsePairs(input)
	if len(cfg.active) > 0 {
		atomic.StoreUint32(&cfg.hasActive, 1)
	}
}

// parsePairs 从 (id1,id2)(id3,id4)... 格式的字符串中解析函数 ID 对。
// 双栏语义：每一对的两个函数互为等待对象，双方都到达后才放行。
func parsePairs(s string) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()

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
		if err == nil {
			cfg.active[id1] = struct{}{}
			cfg.active[id2] = struct{}{}

			gate := &barrierGate{
				id1:     id1,
				id2:     id2,
				release: make(chan struct{}),
			}
			// 双方共享同一个 gate
			cfg.barriers[id1] = append(cfg.barriers[id1], gate)
			cfg.barriers[id2] = append(cfg.barriers[id2], gate)
		}

		s = s[right+1:]
	}
}
