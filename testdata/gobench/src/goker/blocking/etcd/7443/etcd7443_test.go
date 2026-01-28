package etcd7443

import (
	"context"
	sched "sched"
	"sync"
	"testing"
)

type addrConn struct {
	mu    sync.Mutex
	cc    *ClientConn
	addr  Address
	dopts dialOptions
	down  func()
}

func (ac *addrConn) tearDown() {
	sched.InstMutexBF(373662154759, &ac.mu)
	ac.mu.Lock()
	sched.InstMutexAF(373662154759, &ac.mu)
	defer func() {
		sched.InstMutexBF(373662154760, &ac.mu)
		ac.mu.Unlock()
		sched.InstMutexAF(373662154760, &ac.mu)
	}()
	if ac.down != nil {
		ac.down()
		ac.down = nil
	}
}

func (ac *addrConn) resetTransport() {
	sched.InstMutexBF(373662154761, &ac.mu)
	ac.mu.Lock()
	sched.InstMutexAF(373662154761, &ac.mu)
	if ac.cc.dopts.balancer != nil {
		ac.down = ac.cc.dopts.balancer.Up(ac.addr)
	}
	sched.InstMutexBF(373662154762, &ac.mu)
	ac.mu.Unlock()
	sched.InstMutexAF(373662154762, &ac.mu)
}

type ClientConn struct {
	dopts dialOptions
	mu    sync.RWMutex
	conns map[Address]*addrConn
}

func (cc *ClientConn) lbWatcher() {
	for addrs := range cc.dopts.balancer.Notify() {
		var (
			add []Address
			del []*addrConn
		)
		sched.InstMutexBF(373662154763, &cc.mu)
		cc.mu.Lock()
		sched.InstMutexAF(373662154763, &cc.mu)
		for _, a := range addrs {
			if _, ok := cc.conns[a]; !ok {
				add = append(add, a)
			}
		}

		for k, c := range cc.conns {
			var keep bool
			for _, a := range addrs {
				if k == a {
					keep = true
					break
				}
			}
			if !keep {
				del = append(del, c)
				delete(cc.conns, c.addr)
			}
		}
		sched.InstMutexBF(373662154764, &cc.mu)
		cc.mu.Unlock()
		sched.InstMutexAF(373662154764, &cc.mu)
		for _, a := range add {
			cc.resetAddrConn(a)
		}
		for _, c := range del {
			c.tearDown()
		}
	}
}

func (cc *ClientConn) resetAddrConn(addr Address) {
	ac := &addrConn{
		cc:    cc,
		addr:  addr,
		dopts: cc.dopts,
	}
	sched.InstMutexBF(373662154765, &cc.mu)
	cc.mu.Lock()
	sched.InstMutexAF(373662154765, &cc.mu)
	if cc.conns == nil {
		sched.InstMutexBF(373662154766, &cc.mu)
		cc.mu.Unlock()
		sched.InstMutexAF(373662154766, &cc.mu)
		return
	}
	cc.conns[ac.addr] = ac
	sched.InstMutexBF(373662154767, &cc.mu)
	cc.mu.Unlock()
	sched.InstMutexAF(373662154767, &cc.mu)
	go func() {
		ac.resetTransport()
	}()
}

func (cc *ClientConn) Close() {
	sched.InstMutexBF(373662154768, &cc.mu)
	cc.mu.Lock()
	sched.InstMutexAF(373662154768, &cc.mu)
	conns := cc.conns
	cc.conns = nil
	sched.InstMutexBF(373662154769, &cc.mu)
	cc.mu.Unlock()
	sched.InstMutexAF(373662154769, &cc.mu)
	if cc.dopts.balancer != nil {
		cc.dopts.balancer.Close()
	}
	for _, ac := range conns {
		ac.tearDown()
	}
}

type dialOptions struct {
	balancer Balancer
}
type DialOption func(*dialOptions)

func Dial(opts ...DialOption) *ClientConn {
	return DialContext(context.Background(), opts...)
}

func DialContext(ctx context.Context, opts ...DialOption) *ClientConn {
	cc := &ClientConn{
		conns: make(map[Address]*addrConn),
	}
	for _, opt := range opts {
		opt(&cc.dopts)
	}
	go cc.lbWatcher()
	return cc
}

type Balancer interface {
	Up(addr Address) (down func())
	Notify() <-chan []Address
	Close()
}

type Address int

type simpleBalancer struct {
	addrs    []Address
	notifyCh chan []Address
	mu       sync.RWMutex
	closed   bool
	pinAddr  Address
}

func (b *simpleBalancer) Up(addr Address) func() {
	sched.InstMutexBF(373662154770, &b.mu)
	b.mu.Lock()
	sched.InstMutexAF(373662154770, &b.mu)
	defer func() {
		sched.InstMutexBF(373662154771, &b.mu)
		b.mu.Unlock()
		sched.InstMutexAF(373662154771, &b.mu)
	}()

	if b.closed {
		return func() {}
	}

	if b.pinAddr == 0 {
		b.pinAddr = addr
		sched.InstChBF(373662154753, b.notifyCh)
		b.notifyCh <- []Address{addr}
		sched.InstChAF(373662154753, b.notifyCh)
	}

	return func() {
		defer func() {
			if r := recover(); r != nil {
				return
			}
		}()
		sched.InstMutexBF(373662154772, &b.mu)
		b.mu.Lock()
		sched.InstMutexAF(373662154772, &b.mu)
		defer func() {
			sched.InstMutexBF(373662154773, &b.mu)
			b.mu.Unlock()
			sched.InstMutexAF(373662154773, &b.mu)
		}()
		if b.pinAddr == addr {
			b.pinAddr = 0
			sched.InstChBF(373662154754, b.notifyCh)
			b.notifyCh <- b.addrs
			sched.InstChAF(373662154754, b.notifyCh)
		}
	}
}

func (b *simpleBalancer) Notify() <-chan []Address {
	return b.notifyCh
}

func (b *simpleBalancer) Close() {
	sched.InstMutexBF(373662154774, &b.mu)
	b.mu.Lock()
	sched.InstMutexAF(373662154774, &b.mu)
	defer func() {
		sched.InstMutexBF(373662154775, &b.mu)
		b.mu.Unlock()
		sched.InstMutexAF(373662154775, &b.mu)
	}()
	if b.closed {
		return
	}
	b.closed = true
	close(b.notifyCh)
	b.pinAddr = 0
}

func newSimpleBalancer() *simpleBalancer {
	notifyCh := make(chan []Address, 1)
	addrs := make([]Address, 3)
	for i := 0; i < 3; i++ {
		addrs[i] = Address(i)
	}
	sched.InstChBF(373662154756, notifyCh)
	notifyCh <- addrs
	sched.InstChAF(373662154756, notifyCh)
	return &simpleBalancer{
		addrs:    addrs,
		notifyCh: notifyCh,
	}
}

func WithBalancer(b Balancer) DialOption {
	return func(o *dialOptions) {
		o.balancer = b
	}
}
func TestEtcd7443(t *testing.T) {
	sb := newSimpleBalancer()
	conn := Dial(WithBalancer(sb))

	closec := make(chan struct{})
	go func() {
		defer func() {
			sched.InstChBF(373662154757, closec)
			close(closec)
			sched.InstChAF(373662154757, closec)
		}()
		sb.Close()
	}()
	go conn.Close()
	sched.InstChBF(373662154758, closec)
	<-closec
	sched.InstChAF(373662154758, closec)
}
func TestEtcd7443_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	sb := newSimpleBalancer()
	conn := Dial(WithBalancer(sb))

	closec := make(chan struct{})
	go func() {
		defer func() {
			sched.InstChBF(373662154757, closec)
			close(closec)
			sched.InstChAF(373662154757, closec)
		}()
		sb.Close()
	}()
	go conn.Close()
	sched.InstChBF(373662154758, closec)
	<-closec
	sched.InstChAF(373662154758, closec)
}
