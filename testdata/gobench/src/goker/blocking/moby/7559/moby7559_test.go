/*
 * Project: moby
 * Issue or PR  : https://github.com/moby/moby/pull/7559
 * Buggy version: 64579f51fcb439c36377c0068ccc9a007b368b5a
 * fix commit-id: 6cbb8e070d6c3a66bf48fbe5cbf689557eee23db
 * Flaky: 100/100
 */
package moby7559

import (
	"net"
	sched "sched"
	"sync"
	"testing"
)

type UDPProxy struct {
	connTrackLock sync.Mutex
}

func (proxy *UDPProxy) Run() {
	for i := 0; i < 2; i++ {
		sched.InstMutexBF(1043677052929, &proxy.connTrackLock)
		proxy.connTrackLock.Lock()
		sched.InstMutexAF(1043677052929, &proxy.connTrackLock)
		_, err := net.DialUDP("udp", nil, nil)
		if err != nil {
			/// Missing unlock here
			continue
		}
		if i == 0 {
			break
		}
	}
	sched.InstMutexBF(1043677052930, &proxy.connTrackLock)
	proxy.connTrackLock.Unlock()
	sched.InstMutexAF(1043677052930, &proxy.connTrackLock)
}
func TestMoby7559(t *testing.T) {
	proxy := &UDPProxy{}
	go proxy.Run()
}
func TestMoby7559_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	proxy := &UDPProxy{}
	go proxy.Run()
}
