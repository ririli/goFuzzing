package function

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
)

// ParseInput 解析环境变量Input中的调度配置，对标 goroutine.ParseInput。
func ParseInput() {
	input := os.Getenv("Input")
	if input != "" {
		ParsePairs(input)
	}
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()
	if len(cfg.activeMap) > 0 {
		atomic.StoreUint32(&cfg.hasActive, 1)
	}
}

// ParsePairs 解析输入的函数对。
// 格式: (id1,id2)(id3,id4)...
// 双栏语义：每一对的两个函数互为等待对象，双方都到达后才放行。
func ParsePairs(s string) {
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
		if err == nil && id1 != 0 && id2 != 0 && id1 != id2 {
			cfg.activeMap[id1] = struct{}{}
			cfg.activeMap[id2] = struct{}{}

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
