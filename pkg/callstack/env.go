package callstack

import (
	"sync"
)

type Config struct {
	mu sync.RWMutex
	//funcPair          []ConPairFunc              // 当前运行收集的并发调用对
	//suspiciousConPair []SuspiciousConcurrentPair //推测出可能的调用对，后续用断点控制进行验证
	preFuncMap map[uint64][]uint64 // 记录两个函数的前驱
	activeFunc map[uint64]struct{} // 需要验证的函数
	waitMap    sync.Map            // 记录需要等待的函数,value 表示等待的preFunc数量

	hasActive uint32 // 是否有输入
}

func NewConfig() *Config {
	cfg := Config{}
	//cfg.funcPair = make([]ConPairFunc, 0)
	//cfg.suspiciousConPair = make([]SuspiciousConcurrentPair, 0)
	cfg.preFuncMap = make(map[uint64][]uint64)
	cfg.activeFunc = make(map[uint64]struct{})
	return &cfg
}
