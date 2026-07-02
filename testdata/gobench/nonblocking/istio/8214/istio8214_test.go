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
	sched.InstWgBF(369367187460)
	wg.Add(3)
	sched.InstWgAF(369367187460, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(369367187457, _parentGid)
		defer goroutine.Exit(369367187457)
		func() {
			defer func() {
				sched.InstWgBF(369367187461)
				wg.Done()
				sched.InstWgAF(369367187461, &wg, "done")
			}()
			s := &grpcServer{
				cache: &Cache{
					cache: &lruCache{},
				},
			}
			go func(_parentGid uint64) {
				goroutine.Enter(369367187458, _parentGid)
				defer goroutine.Exit(369367187458)
				func() {
					defer func() {
						sched.InstWgBF(369367187462)
						wg.Done()
						sched.InstWgAF(369367187462, &wg, "done")
					}()
					s.Check()
				}()
			}(goroutine.CurrentGid())
			go func(_parentGid uint64) {
				goroutine.Enter(369367187459, _parentGid)
				defer goroutine.Exit(369367187459)
				func() {
					defer func() {
						sched.InstWgBF(369367187463)
						wg.Done()
						sched.InstWgAF(369367187463, &wg, "done")
					}()
					s.Check()
				}()
			}(goroutine.CurrentGid())
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
func TestIstio8214_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	sched.InstWgBF(369367187460)
	wg.Add(3)
	sched.InstWgAF(369367187460, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(369367187457, _parentGid)
		defer goroutine.Exit(369367187457)
		func() {
			defer func() {
				sched.InstWgBF(369367187461)
				wg.Done()
				sched.InstWgAF(369367187461, &wg, "done")
			}()
			s := &grpcServer{
				cache: &Cache{
					cache: &lruCache{},
				},
			}
			go func(_parentGid uint64) {
				goroutine.Enter(369367187458, _parentGid)
				defer goroutine.Exit(369367187458)
				func() {
					defer func() {
						sched.InstWgBF(369367187462)
						wg.Done()
						sched.InstWgAF(369367187462, &wg, "done")
					}()
					s.Check()
				}()
			}(goroutine.CurrentGid())
			go func(_parentGid uint64) {
				goroutine.Enter(369367187459, _parentGid)
				defer goroutine.Exit(369367187459)
				func() {
					defer func() {
						sched.InstWgBF(369367187463)
						wg.Done()
						sched.InstWgAF(369367187463, &wg, "done")
					}()
					s.Check()
				}()
			}(goroutine.CurrentGid())
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
