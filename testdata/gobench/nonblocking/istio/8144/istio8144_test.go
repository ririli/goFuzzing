package istio8144

import (
	"sync"
	"testing"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

type EvictionCallback func()

type callbackRecorder struct {
	callbacks int
}

func (c *callbackRecorder) callback() {
	c.callbacks++
}

type ttlCache struct {
	entries  sync.Map
	callback func()
}

func (c *ttlCache) evicter() {
	c.evictExpired()
}

func (c *ttlCache) evictExpired() {
	c.entries.Range(func(key interface{}, value interface{}) bool {
		c.callback()
		return true
	})
}

func (c *ttlCache) SetWithExpiration(key interface{}, value interface{}) {
	c.entries.Store(key, value)
}

func NewTTLWithCallback(callback EvictionCallback) *ttlCache {
	c := &ttlCache{
		callback: callback,
	}
	go func(_parentGid uint64) {
		goroutine.Enter(4154862224399138817, _parentGid)
		defer goroutine.Exit(4154862224399138817)
		c.evicter()
	}(goroutine.CurrentGid())
	return c
}

func TestIstio8144(t *testing.T) {
	var wg sync.WaitGroup
	sched.InstWgBF(4154862224399138819)
	wg.Add(1)
	sched.InstWgAF(4154862224399138819, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(4154862224399138818, _parentGid)
		defer goroutine.Exit(4154862224399138818)
		func() {
			defer func() {
				sched.InstWgBF(4154862224399138820)
				wg.Done()
				sched.InstWgAF(4154862224399138820, &wg, "done")
			}()
			c := &callbackRecorder{callbacks: 0}
			ttl := NewTTLWithCallback(c.callback)
			ttl.SetWithExpiration(1, 1)
			if c.callbacks != 1 {
			}
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
func TestIstio8144_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	sched.InstWgBF(4154862224399138819)
	wg.Add(1)
	sched.InstWgAF(4154862224399138819, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(4154862224399138818, _parentGid)
		defer goroutine.Exit(4154862224399138818)
		func() {
			defer func() {
				sched.InstWgBF(4154862224399138820)
				wg.Done()
				sched.InstWgAF(4154862224399138820, &wg, "done")
			}()
			c := &callbackRecorder{callbacks: 0}
			ttl := NewTTLWithCallback(c.callback)
			ttl.SetWithExpiration(1, 1)
			if c.callbacks != 1 {
			}
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
