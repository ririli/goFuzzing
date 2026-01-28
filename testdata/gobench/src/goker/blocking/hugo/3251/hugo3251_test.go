package hugo3251

import (
	"fmt"
	sched "sched"
	"sync"
	"testing"
	"time"
)

var (
	remoteURLLock = &remoteLock{m: make(map[string]*sync.Mutex)}
)

type remoteLock struct {
	sync.RWMutex
	m map[string]*sync.Mutex
}

func (l *remoteLock) URLLock(url string) {
	sched.InstMutexBF(25769803777, &l)
	l.Lock()
	sched.InstMutexAF(25769803777, &l)
	if _, ok := l.m[url]; !ok {
		l.m[url] = &sync.Mutex{}
	}
	sched.InstMutexBF(25769803778, &l.m[url])
	l.m[url].Lock()
	sched.InstMutexAF(25769803778, &l.m[url])
	sched.InstMutexBF(25769803779, &l)
	l.Unlock()
	sched.InstMutexAF(25769803779, &l)
}

func (l *remoteLock) URLUnlock(url string) {
	sched.InstMutexBF(25769803780, &l)
	l.RLock()
	sched.InstMutexAF(25769803780, &l)
	defer func() {
		sched.InstMutexBF(25769803781, &l)
		l.RUnlock()
		sched.InstMutexAF(25769803781, &l)
	}()
	if um, ok := l.m[url]; ok {
		sched.InstMutexBF(25769803782, &um)
		um.Unlock()
		sched.InstMutexAF(25769803782, &um)
	}
}

func resGetRemote(url string) error {
	remoteURLLock.URLLock(url)
	defer func() { remoteURLLock.URLUnlock(url) }()

	return nil
}

func TestHugo3251(t *testing.T) {
	url := "http://Foo.Bar/foo_Bar-Foo"
	for _ = range []bool{false, true} {
		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(gor int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					err := resGetRemote(url)
					if err != nil {
						fmt.Errorf("Error getting resource content: %s", err)
					}
					time.Sleep(300 * time.Nanosecond)
				}
			}(i)
		}
		wg.Wait()
	}
}
func TestHugo3251_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	url := "http://Foo.Bar/foo_Bar-Foo"
	for _ = range []bool{false, true} {
		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(gor int) {
				defer wg.Done()
				for j := 0; j < 10; j++ {
					err := resGetRemote(url)
					if err != nil {
						fmt.Errorf("Error getting resource content: %s", err)
					}
					time.Sleep(300 * time.Nanosecond)
				}
			}(i)
		}
		wg.Wait()
	}
}
