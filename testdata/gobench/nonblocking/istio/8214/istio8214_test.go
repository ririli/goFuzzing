package istio8214

import (
	"sync"
	"sync/atomic"
	"testing"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
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
	cc.cache.SetWithExpiration()
	cc.recordStats()
}

func (cc *Cache) recordStats() {
	cc.cache.Stats()
}

type Stats struct {
	Writes uint64
}

type lruCache struct {
	stats Stats
}

func (c *lruCache) Stats() Stats {
	return c.stats
}

func (c *lruCache) Set() {
	c.SetWithExpiration()
}

func (c *lruCache) SetWithExpiration() {
	atomic.AddUint64(&c.stats.Writes, 1)
}

type grpcServer struct {
	cache *Cache
}

func (s *grpcServer) check() {
	if s.cache != nil {
		s.cache.Set()
	}
}

func (s *grpcServer) Check() {
	s.check()
}

func TestIstio8214(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		goroutine.Enter(369367187457)
		defer goroutine.Exit(369367187457)
		func() {
			defer wg.Done()
			s := &grpcServer{
				cache: &Cache{
					cache: &lruCache{},
				},
			}
			go func() {
				goroutine.Enter(369367187458)
				defer goroutine.Exit(369367187458)
				func() {
					defer wg.Done()
					s.Check()
				}()
			}()
			go func() {
				goroutine.Enter(369367187459)
				defer goroutine.Exit(369367187459)
				func() {
					defer wg.Done()
					s.Check()
				}()
			}()
		}()
	}()
	wg.Wait()
}
func TestIstio8214_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		goroutine.Enter(369367187457)
		defer goroutine.Exit(369367187457)
		func() {
			defer wg.Done()
			s := &grpcServer{
				cache: &Cache{
					cache: &lruCache{},
				},
			}
			go func() {
				goroutine.Enter(369367187458)
				defer goroutine.Exit(369367187458)
				func() {
					defer wg.Done()
					s.Check()
				}()
			}()
			go func() {
				goroutine.Enter(369367187459)
				defer goroutine.Exit(369367187459)
				func() {
					defer wg.Done()
					s.Check()
				}()
			}()
		}()
	}()
	wg.Wait()
}
