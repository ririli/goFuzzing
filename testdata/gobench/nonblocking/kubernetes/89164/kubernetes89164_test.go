package kubernetes89164

import (
	"sync"
	"testing"
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
		goroutine.Enter(14954784178565021697, _parentGid)
		defer goroutine.Exit(14954784178565021697)
		cacher.dispatchEvents()
	}(goroutine.CurrentGid())
	return cacher
}

func newTestCacher() *Cacher {
	return NewCacherFromConfig()
}

func TestKubernetes89164(t *testing.T) {
	cacher := newTestCacher()
	for i := 0; i < 3; i++ {
		wg := sync.WaitGroup{}
		sched.InstWgBF(14954784178565021699)
		wg.Add(1)
		sched.InstWgAF(14954784178565021699, &wg, "add")
		go func(_parentGid uint64) {
			goroutine.Enter(14954784178565021698, _parentGid)
			defer goroutine.Exit(14954784178565021698)
			func() {
				cacher.dispatchEvent()
				sched.InstWgBF(14954784178565021700)
				wg.Done()
				sched.InstWgAF(14954784178565021700, &wg, "done")
			}()
		}(goroutine.CurrentGid())
		wg.Wait()
	}
}
func TestKubernetes89164_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	cacher := newTestCacher()
	for i := 0; i < 3; i++ {
		wg := sync.WaitGroup{}
		sched.InstWgBF(14954784178565021699)
		wg.Add(1)
		sched.InstWgAF(14954784178565021699, &wg, "add")
		go func(_parentGid uint64) {
			goroutine.Enter(14954784178565021698, _parentGid)
			defer goroutine.Exit(14954784178565021698)
			func() {
				cacher.dispatchEvent()
				sched.InstWgBF(14954784178565021700)
				wg.Done()
				sched.InstWgAF(14954784178565021700, &wg, "done")
			}()
		}(goroutine.CurrentGid())
		wg.Wait()
	}
}
