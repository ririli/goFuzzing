package etcd4876

import (
	"sync"
	"testing"
	"time"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
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
		goroutine.Enter(820338753537, _parentGid)
		defer goroutine.Exit(820338753537)
		sws.sendLoop()
	}(goroutine.CurrentGid())
}

func TestEtcd4876(t *testing.T) {
	var wg sync.WaitGroup
	sched.InstWgBF(820338753541)
	wg.Add(3)
	sched.InstWgAF(820338753541, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(820338753538, _parentGid)
		defer goroutine.Exit(820338753538)
		func() {
			defer func() {
				sched.InstWgBF(820338753542)
				wg.Done()
				sched.InstWgAF(820338753542, &wg, "done")
			}()
			w := &watchServer{}
			go func(_parentGid uint64) {
				goroutine.Enter(820338753539, _parentGid)
				defer goroutine.Exit(820338753539)
				func() {
					defer func() {
						sched.InstWgBF(820338753543)
						wg.Done()
						sched.InstWgAF(820338753543, &wg, "done")
					}()
					testInterval := 3 * time.Second
					ProgressReportInterval = testInterval
				}()
			}(goroutine.CurrentGid())
			go func(_parentGid uint64) {
				goroutine.Enter(820338753540, _parentGid)
				defer goroutine.Exit(820338753540)
				func() {
					defer func() {
						sched.InstWgBF(820338753544)
						wg.Done()
						sched.InstWgAF(820338753544, &wg, "done")
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
	sched.InstWgBF(820338753541)
	wg.Add(3)
	sched.InstWgAF(820338753541, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(820338753538, _parentGid)
		defer goroutine.Exit(820338753538)
		func() {
			defer func() {
				sched.InstWgBF(820338753542)
				wg.Done()
				sched.InstWgAF(820338753542, &wg, "done")
			}()
			w := &watchServer{}
			go func(_parentGid uint64) {
				goroutine.Enter(820338753539, _parentGid)
				defer goroutine.Exit(820338753539)
				func() {
					defer func() {
						sched.InstWgBF(820338753543)
						wg.Done()
						sched.InstWgAF(820338753543, &wg, "done")
					}()
					testInterval := 3 * time.Second
					ProgressReportInterval = testInterval
				}()
			}(goroutine.CurrentGid())
			go func(_parentGid uint64) {
				goroutine.Enter(820338753540, _parentGid)
				defer goroutine.Exit(820338753540)
				func() {
					defer func() {
						sched.InstWgBF(820338753544)
						wg.Done()
						sched.InstWgAF(820338753544, &wg, "done")
					}()
					w.Watch(&watchWatchServer{})
				}()
			}(goroutine.CurrentGid())
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
