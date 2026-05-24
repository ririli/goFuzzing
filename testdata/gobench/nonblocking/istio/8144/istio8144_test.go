package istio8144

import (
	"sync"
	"testing"
	callstack "toolkit/pkg/callstack"
)

type EvictionCallback func()

type callbackRecorder struct {
	callbacks int
}

func (c *callbackRecorder) callback() {
	defer callstack.Trace(197568495617)()
	c.callbacks++
}

type ttlCache struct {
	entries  sync.Map
	callback func()
}

func (c *ttlCache) evicter() {
	defer callstack.Trace(197568495618)()
	c.evictExpired()
}

func (c *ttlCache) evictExpired() {
	defer callstack.Trace(197568495619)()
	c.entries.Range(func(key interface{}, value interface{}) bool {
		defer callstack.Trace(197568495620)()
		c.callback()
		return true
	})
}

func (c *ttlCache) SetWithExpiration(key interface{}, value interface{}) {
	defer callstack.Trace(197568495621)()
	c.entries.Store(key, value)
}

func NewTTLWithCallback(callback EvictionCallback) *ttlCache {
	defer callstack.Trace(197568495622)()
	c := &ttlCache{
		callback: callback,
	}
	go c.evicter()
	return c
}

func TestIstio8144(t *testing.T) {
	defer callstack.Trace(197568495623)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(197568495624)()
		defer wg.Done()
		c := &callbackRecorder{callbacks: 0}
		ttl := NewTTLWithCallback(c.callback)
		ttl.SetWithExpiration(1, 1)
		if c.callbacks != 1 {
		}
	}()
	wg.Wait()
}
func TestIstio8144_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(197568495623)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(197568495624)()
		defer wg.Done()
		c := &callbackRecorder{callbacks: 0}
		ttl := NewTTLWithCallback(c.callback)
		ttl.SetWithExpiration(1, 1)
		if c.callbacks != 1 {
		}
	}()
	wg.Wait()
}
