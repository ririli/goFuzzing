package grpc1748

import (
	"sync"
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
)

var minConnectTimeout = 10 * time.Second

var balanceMutex sync.Mutex // We add this for avoiding other data race

type Balancer interface {
	HandleResolvedAddrs()
}

type Builder interface {
	Build(cc balancer_ClientConn) Balancer
}

func newPickfirstBuilder() Builder {
	defer callstack.Trace(987842478081)()
	return &pickfirstBuilder{}
}

type pickfirstBuilder struct{}

func (*pickfirstBuilder) Build(cc balancer_ClientConn) Balancer {
	defer callstack.Trace(987842478082)()
	return &pickfirstBalancer{cc: cc}
}

type SubConn interface {
	Connect()
}

type balancer_ClientConn interface {
	NewSubConn() SubConn
}

type pickfirstBalancer struct {
	cc balancer_ClientConn
	sc SubConn
}

func (b *pickfirstBalancer) HandleResolvedAddrs() {
	defer callstack.Trace(987842478083)()
	b.sc = b.cc.NewSubConn()
	b.sc.Connect()
}

type pickerWrapper struct {
	mu sync.Mutex
}

type acBalancerWrapper struct {
	mu sync.Mutex
	ac *addrConn
}

type addrConn struct {
	cc   *ClientConn
	acbw SubConn
	mu   sync.Mutex
}

func (ac *addrConn) resetTransport() {
	defer callstack.Trace(987842478084)()
	_ = minConnectTimeout
}

func (ac *addrConn) transportMonitor() {
	defer callstack.Trace(987842478085)()
	ac.resetTransport()
}

func (ac *addrConn) connect() {
	defer callstack.Trace(987842478086)()
	go func() {
		defer callstack.Trace(987842478087)()
		ac.transportMonitor()
	}()
}

func (acbw *acBalancerWrapper) Connect() {
	defer callstack.Trace(987842478088)()
	acbw.mu.Lock()
	defer acbw.mu.Unlock()
	acbw.ac.connect()
}

func newPickerWrapper() *pickerWrapper {
	defer callstack.Trace(987842478089)()
	return &pickerWrapper{}
}

type ClientConn struct {
	mu sync.Mutex
}

func (cc *ClientConn) switchBalancer() {
	defer callstack.Trace(987842478090)()
	builder := newPickfirstBuilder()
	newCCBalancerWrapper(cc, builder)
}

func (cc *ClientConn) newAddrConn() *addrConn {
	defer callstack.Trace(987842478091)()
	return &addrConn{cc: cc}
}

type ccBalancerWrapper struct {
	cc       *ClientConn
	balancer Balancer
}

func (ccb *ccBalancerWrapper) watcher() {
	defer callstack.Trace(987842478092)()
	for i := 0; i < 10; i++ {
		balanceMutex.Lock()
		if ccb.balancer != nil {
			balanceMutex.Unlock()
			ccb.balancer.HandleResolvedAddrs()
		} else {
			balanceMutex.Unlock()
		}
	}
}

func (ccb *ccBalancerWrapper) NewSubConn() SubConn {
	defer callstack.Trace(987842478093)()
	ac := ccb.cc.newAddrConn()
	acbw := &acBalancerWrapper{ac: ac}
	acbw.ac.mu.Lock()
	ac.acbw = acbw
	acbw.ac.mu.Unlock()
	return acbw
}

func newCCBalancerWrapper(cc *ClientConn, b Builder) {
	defer callstack.Trace(987842478094)()
	ccb := &ccBalancerWrapper{cc: cc}
	go ccb.watcher()
	balanceMutex.Lock()
	defer balanceMutex.Unlock()
	ccb.balancer = b.Build(ccb)
}

func TestGrpc1748(t *testing.T) {
	defer callstack.Trace(987842478095)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(987842478096)()
		defer wg.Done()
		mctBkp := minConnectTimeout
		// Call this only after transportMonitor goroutine has ended.
		defer func() {
			defer callstack.Trace(987842478097)()
			minConnectTimeout = mctBkp
		}()
		cc := &ClientConn{}
		cc.switchBalancer()
	}()
	wg.Wait()
}
func TestGrpc1748_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(987842478095)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(987842478096)()
		defer wg.Done()
		mctBkp := minConnectTimeout

		defer func() {
			defer callstack.Trace(987842478097)()
			minConnectTimeout = mctBkp
		}()
		cc := &ClientConn{}
		cc.switchBalancer()
	}()
	wg.Wait()
}
