package etcd4876

import (
	"sync"
	"testing"
	"time"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/operation"
)

var ProgressReportInterval = 10 * time.Second

type Watcher interface {
	Watch()
}
type ServerStream interface{}

type Watch_WatchServer interface {
	Send()
	ServerStream
}
type watchWatchServer struct {
	ServerStream
}

func (x *watchWatchServer) Send() {}

type WatchServer interface {
	Watch(Watch_WatchServer)
}

type serverWatchStream struct{}

func (sws *serverWatchStream) sendLoop() {
	_ = time.NewTicker(ProgressReportInterval)
}

type watchServer struct{}

func (ws *watchServer) Watch(stream Watch_WatchServer) {
	//inst.WaitTimeout(500 * time.Millisecond)
	sws := serverWatchStream{}
	go func(_parentGid uint64) {
		goroutine.Enter(7814497140268335105, _parentGid)
		defer goroutine.Exit(7814497140268335105)
		sws.sendLoop()
	}(goroutine.CurrentGid())
}

func TestEtcd4876(t *testing.T) {
	var wg sync.WaitGroup
	sched.InstWgBF(7814497140268335109)
	wg.Add(3)
	sched.InstWgAF(7814497140268335109, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(7814497140268335106, _parentGid)
		defer goroutine.Exit(7814497140268335106)
		func() {
			defer func() {
				sched.InstWgBF(7814497140268335110)
				wg.Done()
				sched.InstWgAF(7814497140268335110, &wg, "done")
			}()
			w := &watchServer{}
			go func(_parentGid uint64) {
				goroutine.Enter(7814497140268335107, _parentGid)
				defer goroutine.Exit(7814497140268335107)
				func() {
					defer func() {
						sched.InstWgBF(7814497140268335111)
						wg.Done()
						sched.InstWgAF(7814497140268335111, &wg, "done")
					}()
					testInterval := 3 * time.Second
					ProgressReportInterval = testInterval
				}()
			}(goroutine.CurrentGid())
			go func(_parentGid uint64) {
				goroutine.Enter(7814497140268335108, _parentGid)
				defer goroutine.Exit(7814497140268335108)
				func() {
					defer func() {
						sched.InstWgBF(7814497140268335112)
						wg.Done()
						sched.InstWgAF(7814497140268335112, &wg, "done")
					}()
					w.Watch(&watchWatchServer{})
				}()
			}(goroutine.CurrentGid())
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
func TestEtcd4876_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	sched.InstWgBF(7814497140268335109)
	wg.Add(3)
	sched.InstWgAF(7814497140268335109, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(7814497140268335106, _parentGid)
		defer goroutine.Exit(7814497140268335106)
		func() {
			defer func() {
				sched.InstWgBF(7814497140268335110)
				wg.Done()
				sched.InstWgAF(7814497140268335110, &wg, "done")
			}()
			w := &watchServer{}
			go func(_parentGid uint64) {
				goroutine.Enter(7814497140268335107, _parentGid)
				defer goroutine.Exit(7814497140268335107)
				func() {
					defer func() {
						sched.InstWgBF(7814497140268335111)
						wg.Done()
						sched.InstWgAF(7814497140268335111, &wg, "done")
					}()
					testInterval := 3 * time.Second
					ProgressReportInterval = testInterval
				}()
			}(goroutine.CurrentGid())
			go func(_parentGid uint64) {
				goroutine.Enter(7814497140268335108, _parentGid)
				defer goroutine.Exit(7814497140268335108)
				func() {
					defer func() {
						sched.InstWgBF(7814497140268335112)
						wg.Done()
						sched.InstWgAF(7814497140268335112, &wg, "done")
					}()
					w.Watch(&watchWatchServer{})
				}()
			}(goroutine.CurrentGid())
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
