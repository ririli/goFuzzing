package kubernetes89164

import (
	"sync"
	"testing"
	callstack "toolkit/pkg/callstack"
)

type cacheWatcher int

type Cacher struct {
	sync.RWMutex
	watcherBuffer []*cacheWatcher
}

func (c *Cacher) startDispatching() {
	defer callstack.Trace(429496729601)()
	c.Lock()
	defer c.Unlock()

	c.watcherBuffer = c.watcherBuffer[:0]
}

func (c *Cacher) dispatchEvent() {
	defer callstack.Trace(429496729602)()
	c.startDispatching()
	for _ = range c.watcherBuffer {
	}
}

func (c *Cacher) dispatchEvents() {
	defer callstack.Trace(429496729603)()
	c.dispatchEvent()
}

func NewCacherFromConfig() *Cacher {
	defer callstack.Trace(429496729604)()
	cacher := &Cacher{}
	go cacher.dispatchEvents()
	return cacher
}

func newTestCacher() *Cacher {
	defer callstack.Trace(429496729605)()
	return NewCacherFromConfig()
}

func TestKubernetes89164(t *testing.T) {
	defer callstack.Trace(429496729606)()
	cacher := newTestCacher()
	for i := 0; i < 3; i++ {
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer callstack.Trace(429496729607)()
			cacher.dispatchEvent()
			wg.Done()
		}()
		wg.Wait()
	}
}
func TestKubernetes89164_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.Trace(429496729606)()
	cacher := newTestCacher()
	for i := 0; i < 3; i++ {
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer callstack.Trace(429496729607)()
			cacher.dispatchEvent()
			wg.Done()
		}()
		wg.Wait()
	}
}
