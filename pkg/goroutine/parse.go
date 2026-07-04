package goroutine

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
)

// ParseInput 解析环境变量Input中的调度配置
func ParseInput() {
	input_susPairs := os.Getenv("Input")
	if input_susPairs != "" {
		ParsePairs(input_susPairs)
	}
	if len(cfg.activeMap) > 0 {
		atomic.StoreUint32(&cfg.hasActive, 1)
	}
}

// ParsePairs 解析输入的goroutine对
// 格式: (id1,id2)(id3,id4)(id5,id6)...
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
		if err == nil {
			cfg.activeMap[id1] = struct{}{}
			cfg.activeMap[id2] = struct{}{}
			cfg.preMap[id2] = append(cfg.preMap[id2], id1)
			actual, _ := cfg.waitMap.LoadOrStore(id2, new(atomic.Int32))
			actual.(*atomic.Int32).Add(1)
		}

		s = s[right+1:]
	}
}
