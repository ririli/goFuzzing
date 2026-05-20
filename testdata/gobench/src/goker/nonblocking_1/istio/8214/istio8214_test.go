package istio8214

import (
	"sync"
	"sync/atomic"
	"testing"
	callstack "toolkit/pkg/callstack"
)

type internal_Cache interface {
	Set()
	Stats() Stats
}

type ExpiringCache interface {
	internal_Cache
	SetWithExpiration()
}

type Cache struct {
	cache ExpiringCache
}

func (cc *Cache) Set() {
	defer callstack.Trace(408021893121)()
	cc.cache.SetWithExpiration()
	cc.recordStats()
}

func (cc *Cache) recordStats() {
	defer callstack.Trace(408021893122)()
	cc.cache.Stats()
}

type Stats struct {
	Writes uint64
}

type lruCache struct {
	stats Stats
}

func (c *lruCache) Stats() Stats {
	defer callstack.Trace(408021893123)()
	return c.stats
}

func (c *lruCache) Set() {
	defer callstack.Trace(408021893124)()
	c.SetWithExpiration()
}

func (c *lruCache) SetWithExpiration() {
	defer callstack.Trace(408021893125)()
	atomic.AddUint64(&c.stats.Writes, 1)
}

type grpcServer struct {
	cache *Cache
}

func (s *grpcServer) check() {
	defer callstack.Trace(408021893126)()
	if s.cache != nil {
		s.cache.Set()
	}
}

func (s *grpcServer) Check() {
	defer callstack.Trace(408021893127)()
	s.check()
}

func TestIstio8214(t *testing.T) {
	defer callstack.Trace(408021893128)()
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer callstack.Trace(408021893129)()
		defer wg.Done()
		s := &grpcServer{
			cache: &Cache{
				cache: &lruCache{},
			},
		}
		go func() {
			defer callstack.Trace(408021893130)()
			defer wg.Done()
			s.Check()
		}()
		go func() {
			defer callstack.Trace(408021893131)()
			defer wg.Done()
			s.Check()
		}()
	}()
	wg.Wait()
}
func TestIstio8214_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.Trace(408021893128)()
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer callstack.Trace(408021893129)()
		defer wg.Done()
		s := &grpcServer{
			cache: &Cache{
				cache: &lruCache{},
			},
		}
		go func() {
			defer callstack.Trace(408021893130)()
			defer wg.Done()
			s.Check()
		}()
		go func() {
			defer callstack.Trace(408021893131)()
			defer wg.Done()
			s.Check()
		}()
	}()
	wg.Wait()
}
