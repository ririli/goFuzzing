package goroutine

import (
	"sync"
	"time"
)

// Config 记录goroutine调度配置
type Config struct {
	mu sync.RWMutex
	// barriers 记录goroutine之间的双栏同步门：goroutineId → 该gid参与的所有barrierGate
	barriers map[uint64][]*barrierGate
	// activeMap 记录需要调度的goroutine集合
	activeMap map[uint64]struct{}
	// hasActive 是否有输入
	hasActive uint32
	// BarrierTimeout 单个 barrier 等待超时时间
	BarrierTimeout time.Duration
}

func NewConfig() *Config {
	cfg1 := Config{}
	cfg1.barriers = make(map[uint64][]*barrierGate)
	cfg1.activeMap = make(map[uint64]struct{})
	cfg1.BarrierTimeout = 10 * time.Millisecond
	return &cfg1
}
