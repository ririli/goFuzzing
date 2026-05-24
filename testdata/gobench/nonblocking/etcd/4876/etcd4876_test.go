package etcd4876

import (
	"sync"
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
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
	defer callstack.Trace(820338753537)()
	_ = time.NewTicker(ProgressReportInterval)
}

type watchServer struct{}

func (ws *watchServer) Watch(stream Watch_WatchServer) {
	defer callstack.Trace(820338753538)()
	sws := serverWatchStream{}
	go sws.sendLoop()
}

func TestEtcd4876(t *testing.T) {
	defer callstack.Trace(820338753539)()
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer callstack.Trace(820338753540)()
		defer wg.Done()
		w := &watchServer{}
		go func() {
			defer callstack.Trace(820338753541)()
			defer wg.Done()
			testInterval := 3 * time.Second
			ProgressReportInterval = testInterval
		}()
		go func() {
			defer callstack.Trace(820338753542)()
			defer wg.Done()
			w.Watch(&watchWatchServer{})
		}()
	}()
	wg.Wait()
}
func TestEtcd4876_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(820338753539)()
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer callstack.Trace(820338753540)()
		defer wg.Done()
		w := &watchServer{}
		go func() {
			defer callstack.Trace(820338753541)()
			defer wg.Done()
			testInterval := 3 * time.Second
			ProgressReportInterval = testInterval
		}()
		go func() {
			defer callstack.Trace(820338753542)()
			defer wg.Done()
			w.Watch(&watchWatchServer{})
		}()
	}()
	wg.Wait()
}
