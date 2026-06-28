// Package breakpoint 提供函数对之间的断点/等待控制机制。
// 用于在运行时验证可疑的并发函数对 —— 让函数 B 等待函数 A 完成后才继续执行。
package breakpoint

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Config 断点控制配置
type Config struct {
	mu sync.RWMutex
	// preFuncMap 记录函数的前驱关系：key 的后驱是 value 列表中的函数
	preFuncMap map[uint64][]uint64
	// activeFunc 需要验证的函数集合
	activeFunc map[uint64]struct{}
	// waitMap 记录需要等待的函数，value 表示等待的 preFunc 数量
	waitMap sync.Map
	// hasActive 是否有输入（原子操作）
	hasActive uint32
	// Timeout 等待超时时间
	Timeout time.Duration
	// Waiters channel 池，用于 goroutine 间同步
	Waiters sync.Map
}

// NewConfig 创建新的断点控制配置
func NewConfig() *Config {
	return &Config{
		preFuncMap: make(map[uint64][]uint64),
		activeFunc: make(map[uint64]struct{}),
		Timeout:    40 * time.Millisecond,
	}
}

// ParseInput 从环境变量 Input 解析可疑函数对并设置活动状态
func (c *Config) ParseInput() {
	inputSusPairs := os.Getenv("Input")
	if inputSusPairs != "" {
		c.ParseSusPairs(inputSusPairs)
	}
	if len(c.activeFunc) > 0 {
		atomic.StoreUint32(&c.hasActive, 1)
	}
}

// ParseSusPairs 解析输入的函数对
// 格式: (id1,id2)(id3,id4)(id5,id6)...
// id2 需要等待 id1 完成后才能执行
func (c *Config) ParseSusPairs(s string) {
	c.mu.Lock()
	defer c.mu.Unlock()

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
			c.activeFunc[id1] = struct{}{}
			c.activeFunc[id2] = struct{}{}
			c.preFuncMap[id2] = append(c.preFuncMap[id2], id1)
			actual, _ := c.waitMap.LoadOrStore(id2, new(atomic.Int32))
			actual.(*atomic.Int32).Add(1)
		}

		s = s[right+1:]
	}
}

// HasActive 返回是否有活动的断点配置
func (c *Config) HasActive() bool {
	return atomic.LoadUint32(&c.hasActive) == 1
}

// FindPrev 查找指定 funcId 的前驱 ID 列表
func (c *Config) FindPrev(funcId uint64) []uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if prevId, ok := c.preFuncMap[funcId]; ok {
		return prevId
	}
	return nil
}

// IsActive 判断函数是否处于活动状态
func (c *Config) IsActive(funcId uint64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.activeFunc[funcId]
	return ok
}

// DoWait 判断函数是否需要等待前驱完成
func (c *Config) DoWait(funcId uint64) bool {
	if value, ok := c.waitMap.Load(funcId); ok {
		return value.(*atomic.Int32).Load() > 0
	}
	return false
}

// WaitMapDec 减少指定函数 ID 的等待计数
func (c *Config) WaitMapDec(funcId uint64) {
	if val, ok := c.waitMap.Load(funcId); ok {
		newVal := val.(*atomic.Int32).Add(-1)
		if newVal <= 0 {
			c.waitMap.Delete(funcId)
		}
	}
}
