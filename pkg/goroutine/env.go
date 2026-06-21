package goroutine

import (
	"sync"
)

// Config 记录goroutine调度配置
type Config struct {
	mu sync.RWMutex
	// preMap 记录goroutine之间的前驱关系: goroutineId -> 需要等待的前驱goroutineId列表
	preMap map[uint64][]uint64
	// activeMap 记录需要调度的goroutine集合
	activeMap map[uint64]struct{}
	// waitMap 记录需要等待的goroutine, value表示等待的前驱goroutine数量
	waitMap sync.Map // goroutineId -> *atomic.Int32

	hasActive uint32 // 是否有输入
}

func NewConfig() *Config {
	cfg := Config{}
	cfg.preMap = make(map[uint64][]uint64)
	cfg.activeMap = make(map[uint64]struct{})
	return &cfg
}
