package grpc1748

import (
	"sync"
	"testing"
	"time"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/operation"
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
	return &pickfirstBuilder{}
}

type pickfirstBuilder struct{}

func (*pickfirstBuilder) Build(cc balancer_ClientConn) Balancer {
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
	_ = minConnectTimeout
}

func (ac *addrConn) transportMonitor() {
	ac.resetTransport()
}

func (ac *addrConn) connect() {
	go func(_parentGid uint64) {
		goroutine.Enter(18253461912505286657, _parentGid)
		defer goroutine.Exit(18253461912505286657)
		func() {
			ac.transportMonitor()
		}()
	}(goroutine.CurrentGid())
}

func (acbw *acBalancerWrapper) Connect() {
	acbw.mu.Lock()
	defer acbw.mu.Unlock()
	acbw.ac.connect()
}

func newPickerWrapper() *pickerWrapper {
	return &pickerWrapper{}
}

type ClientConn struct {
	mu sync.Mutex
}

func (cc *ClientConn) switchBalancer() {
	builder := newPickfirstBuilder()
	newCCBalancerWrapper(cc, builder)
}

func (cc *ClientConn) newAddrConn() *addrConn {
	return &addrConn{cc: cc}
}

type ccBalancerWrapper struct {
	cc       *ClientConn
	balancer Balancer
}

func (ccb *ccBalancerWrapper) watcher() {
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
	ac := ccb.cc.newAddrConn()
	acbw := &acBalancerWrapper{ac: ac}
	acbw.ac.mu.Lock()
	ac.acbw = acbw
	acbw.ac.mu.Unlock()
	return acbw
}

func newCCBalancerWrapper(cc *ClientConn, b Builder) {
	ccb := &ccBalancerWrapper{cc: cc}
	go func(_parentGid uint64) {
		goroutine.Enter(18253461912505286658, _parentGid)
		defer goroutine.Exit(18253461912505286658)
		ccb.watcher()
	}(goroutine.CurrentGid())
	balanceMutex.Lock()
	defer balanceMutex.Unlock()
	ccb.balancer = b.Build(ccb)
}

func TestGrpc1748(t *testing.T) {
	var wg sync.WaitGroup
	sched.InstWgBF(18253461912505286660)
	wg.Add(1)
	sched.InstWgAF(18253461912505286660, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(18253461912505286659, _parentGid)

		// Call this only after transportMonitor goroutine has ended.
		defer goroutine.Exit(18253461912505286659)
		func() {
			defer func() {
				sched.InstWgBF(18253461912505286661)
				wg.Done()
				sched.InstWgAF(18253461912505286661, &wg, "done")
			}()
			mctBkp := minConnectTimeout

			defer func() {
				minConnectTimeout = mctBkp
			}()
			cc := &ClientConn{}
			cc.switchBalancer()
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
func TestGrpc1748_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	sched.InstWgBF(18253461912505286660)
	wg.Add(1)
	sched.InstWgAF(18253461912505286660, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(18253461912505286659, _parentGid)
		defer goroutine.Exit(18253461912505286659)
		func() {
			defer func() {
				sched.InstWgBF(18253461912505286661)
				wg.Done()
				sched.InstWgAF(18253461912505286661, &wg, "done")
			}()
			mctBkp := minConnectTimeout

			defer func() {
				minConnectTimeout = mctBkp
			}()
			cc := &ClientConn{}
			cc.switchBalancer()
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
