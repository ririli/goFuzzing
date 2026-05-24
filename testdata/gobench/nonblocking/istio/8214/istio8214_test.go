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
	defer callstack.Trace(369367187457)()
	cc.cache.SetWithExpiration()
	cc.recordStats()
}

func (cc *Cache) recordStats() {
	defer callstack.Trace(369367187458)()
	cc.cache.Stats()
}

type Stats struct {
	Writes uint64
}

type lruCache struct {
	stats Stats
}

func (c *lruCache) Stats() Stats {
	defer callstack.Trace(369367187459)()
	return c.stats
}

func (c *lruCache) Set() {
	defer callstack.Trace(369367187460)()
	c.SetWithExpiration()
}

func (c *lruCache) SetWithExpiration() {
	defer callstack.Trace(369367187461)()
	atomic.AddUint64(&c.stats.Writes, 1)
}

type grpcServer struct {
	cache *Cache
}

func (s *grpcServer) check() {
	defer callstack.Trace(369367187462)()
	if s.cache != nil {
		s.cache.Set()
	}
}

func (s *grpcServer) Check() {
	defer callstack.Trace(369367187463)()
	s.check()
}

func TestIstio8214(t *testing.T) {
	defer callstack.Trace(369367187464)()
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer callstack.Trace(369367187465)()
		defer wg.Done()
		s := &grpcServer{
			cache: &Cache{
				cache: &lruCache{},
			},
		}
		go func() {
			defer callstack.Trace(369367187466)()
			defer wg.Done()
			s.Check()
		}()
		go func() {
			defer callstack.Trace(369367187467)()
			defer wg.Done()
			s.Check()
		}()
	}()
	wg.Wait()
}
func TestIstio8214_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(369367187464)()
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer callstack.Trace(369367187465)()
		defer wg.Done()
		s := &grpcServer{
			cache: &Cache{
				cache: &lruCache{},
			},
		}
		go func() {
			defer callstack.Trace(369367187466)()
			defer wg.Done()
			s.Check()
		}()
		go func() {
			defer callstack.Trace(369367187467)()
			defer wg.Done()
			s.Check()
		}()
	}()
	wg.Wait()
}
