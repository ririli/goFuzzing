package kubernetes77796

import (
	"sync"
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
)

type cacheWatcher int

type Cacher struct {
	sync.RWMutex
	watcherBuffer []*cacheWatcher
}

func (c *Cacher) startDispatching() {
	defer callstack.Trace(180388626433)()
	c.Lock()
	defer c.Unlock()

	c.watcherBuffer = c.watcherBuffer[:0]
}

func (c *Cacher) dispatchEvent() {
	defer callstack.Trace(180388626434)()
	c.startDispatching()
	for _ = range c.watcherBuffer {
	}
}

func (c *Cacher) dispatchEvents() {
	defer callstack.Trace(180388626435)()
	c.dispatchEvent()
}

func NewCacherFromConfig() *Cacher {
	defer callstack.Trace(180388626436)()
	cacher := &Cacher{}
	go cacher.dispatchEvents()
	return cacher
}

func newTestCacher() *Cacher {
	defer callstack.Trace(180388626437)()
	return NewCacherFromConfig()
}

func TestKubernetes77796(t *testing.T) {
	defer callstack.Trace(180388626438)()
	cacher := newTestCacher()
	for i := 0; i < 3; i++ {
		go func() {
			defer callstack.Trace(180388626439)()
			cacher.dispatchEvent()
		}()
		time.Sleep(10 * time.Millisecond)
	}
}
func TestKubernetes77796_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.Trace(180388626438)()
	cacher := newTestCacher()
	for i := 0; i < 3; i++ {
		go func() {
			defer callstack.Trace(180388626439)()
			cacher.dispatchEvent()
		}()
		time.Sleep(10 * time.Millisecond)
	}
}
