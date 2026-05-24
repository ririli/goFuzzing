package grpc2371

import (
	"sync"
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
)

type ccBalancerWrapper struct {
	cc               *ClientConn
	resolverUpdateCh chan struct{}
}

func (ccb *ccBalancerWrapper) handleResolvedAddrs() {
	defer callstack.Trace(1022202216449)()
	select {
	case <-ccb.resolverUpdateCh:
	default:
	}
	ccb.resolverUpdateCh <- struct{}{}
}

func newCCBalancerWrapper(cc *ClientConn) *ccBalancerWrapper {
	defer callstack.Trace(1022202216450)()
	ccb := &ccBalancerWrapper{
		cc:               cc,
		resolverUpdateCh: make(chan struct{}, 1),
	}
	return ccb
}

type ccResolverWrapper struct {
	cc *ClientConn
}

func (ccr *ccResolverWrapper) start() {
	defer callstack.Trace(1022202216451)()
	go ccr.watcher()
}

func (ccr *ccResolverWrapper) watcher() {
	defer callstack.Trace(1022202216452)()
	ccr.cc.handleServiceConfig()
}

func newCCResolverWrapper(cc *ClientConn) *ccResolverWrapper {
	defer callstack.Trace(1022202216453)()
	ccr := &ccResolverWrapper{
		cc: cc,
	}
	return ccr
}

type ClientConn struct {
	mu              sync.RWMutex
	balancerWrapper *ccBalancerWrapper
	resolverWrapper *ccResolverWrapper
}

func (cc *ClientConn) handleServiceConfig() {
	defer callstack.Trace(1022202216454)()
	cc.mu.Lock()
	cc.balancerWrapper.handleResolvedAddrs()
	cc.mu.Unlock()
}

func (cc *ClientConn) Close() {
	defer callstack.Trace(1022202216455)()
	cc.mu.Lock()
	cc.resolverWrapper = nil
	cc.balancerWrapper = nil
	cc.mu.Unlock()
}

func Dial() *ClientConn {
	defer callstack.Trace(1022202216456)()
	return DialContext()
}

func DialContext() *ClientConn {
	defer callstack.Trace(1022202216457)()
	cc := &ClientConn{}

	cc.resolverWrapper = newCCResolverWrapper(cc)
	cc.balancerWrapper = newCCBalancerWrapper(cc)

	cc.resolverWrapper.start()

	return cc
}

func TestGrpc2371(t *testing.T) {
	defer callstack.Trace(1022202216458)()

	for i := 0; i < 10; i++ {
		cc := Dial()

		go cc.Close()
	}

	time.Sleep(100 * time.Millisecond)
}
func TestGrpc2371_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(1022202216458)()
	for i := 0; i < 10; i++ {
		cc := Dial()

		go cc.Close()
	}

	time.Sleep(100 * time.Millisecond)
}
