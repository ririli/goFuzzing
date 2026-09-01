package operation

import "sync"

type Config struct {
	waitMap  map[uint64]uint64
	preOpMap map[uint64][]uint64
	active   map[uint64]struct{}
	mu       sync.RWMutex
}

func NewConfig() *Config {
	config := Config{}
	config.preOpMap = make(map[uint64][]uint64)
	config.waitMap = make(map[uint64]uint64)
	config.active = make(map[uint64]struct{})
	return &config
}
