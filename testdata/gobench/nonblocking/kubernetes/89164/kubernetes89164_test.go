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
	defer callstack.Trace(1073741824001)()
	c.Lock()
	defer c.Unlock()

	c.watcherBuffer = c.watcherBuffer[:0]
}

func (c *Cacher) dispatchEvent() {
	defer callstack.Trace(1073741824002)()
	c.startDispatching()
	for _ = range c.watcherBuffer {
	}
}

func (c *Cacher) dispatchEvents() {
	defer callstack.Trace(1073741824003)()
	c.dispatchEvent()
}

func NewCacherFromConfig() *Cacher {
	defer callstack.Trace(1073741824004)()
	cacher := &Cacher{}
	go cacher.dispatchEvents()
	return cacher
}

func newTestCacher() *Cacher {
	defer callstack.Trace(1073741824005)()
	return NewCacherFromConfig()
}

func TestKubernetes89164(t *testing.T) {
	defer callstack.Trace(1073741824006)()
	cacher := newTestCacher()
	for i := 0; i < 3; i++ {
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer callstack.Trace(1073741824007)()
			cacher.dispatchEvent()
			wg.Done()
		}()
		wg.Wait()
	}
}
func TestKubernetes89164_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(1073741824006)()
	cacher := newTestCacher()
	for i := 0; i < 3; i++ {
		wg := sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer callstack.Trace(1073741824007)()
			cacher.dispatchEvent()
			wg.Done()
		}()
		wg.Wait()
	}
}
