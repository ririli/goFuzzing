package kubernetes77796

import (
	"sync"
	"testing"
	"time"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

type cacheWatcher int

type Cacher struct {
	sync.RWMutex
	watcherBuffer []*cacheWatcher
}

func (c *Cacher) startDispatching() {
	c.Lock()
	defer c.Unlock()

	c.watcherBuffer = c.watcherBuffer[:0]
}

func (c *Cacher) dispatchEvent() {
	c.startDispatching()
	for _ = range c.watcherBuffer {
	}
}

func (c *Cacher) dispatchEvents() {
	c.dispatchEvent()
}

func NewCacherFromConfig() *Cacher {
	cacher := &Cacher{}
	go func(_parentGid uint64) {
		goroutine.Enter(15268316082503417857, _parentGid)
		defer goroutine.Exit(15268316082503417857)
		cacher.dispatchEvents()
	}(goroutine.CurrentGid())
	return cacher
}

func newTestCacher() *Cacher {
	return NewCacherFromConfig()
}

func TestKubernetes77796(t *testing.T) {
	cacher := newTestCacher()
	for i := 0; i < 3; i++ {
		go func(_parentGid uint64) {
			goroutine.Enter(15268316082503417858, _parentGid)
			defer goroutine.Exit(15268316082503417858)
			func() {
				cacher.dispatchEvent()
			}()
		}(goroutine.CurrentGid())
		time.Sleep(10 * time.Millisecond)
	}
}
func TestKubernetes77796_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	cacher := newTestCacher()
	for i := 0; i < 3; i++ {
		go func(_parentGid uint64) {
			goroutine.Enter(15268316082503417858, _parentGid)
			defer goroutine.Exit(15268316082503417858)
			func() {
				cacher.dispatchEvent()
			}()
		}(goroutine.CurrentGid())
		time.Sleep(10 * time.Millisecond)
	}
}
