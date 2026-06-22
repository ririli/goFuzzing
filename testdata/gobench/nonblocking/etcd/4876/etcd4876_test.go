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
	go func() {
		goroutine.Enter(820338753537)
		defer goroutine.Exit(820338753537)
		sws.sendLoop()
	}()
}

func TestEtcd4876(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		goroutine.Enter(820338753538)
		defer goroutine.Exit(820338753538)
		func() {
			defer wg.Done()
			w := &watchServer{}
			go func() {
				goroutine.Enter(820338753539)
				defer goroutine.Exit(820338753539)
				func() {
					defer wg.Done()
					testInterval := 3 * time.Second
					ProgressReportInterval = testInterval
				}()
			}()
			go func() {
				goroutine.Enter(820338753540)
				defer goroutine.Exit(820338753540)
				func() {
					defer wg.Done()
					w.Watch(&watchWatchServer{})
				}()
			}()
		}()
	}()
	wg.Wait()
}
func TestEtcd4876_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		goroutine.Enter(820338753538)
		defer goroutine.Exit(820338753538)
		func() {
			defer wg.Done()
			w := &watchServer{}
			go func() {
				goroutine.Enter(820338753539)
				defer goroutine.Exit(820338753539)
				func() {
					defer wg.Done()
					testInterval := 3 * time.Second
					ProgressReportInterval = testInterval
				}()
			}()
			go func() {
				goroutine.Enter(820338753540)
				defer goroutine.Exit(820338753540)
				func() {
					defer wg.Done()
					w.Watch(&watchWatchServer{})
				}()
			}()
		}()
	}()
	wg.Wait()
}
