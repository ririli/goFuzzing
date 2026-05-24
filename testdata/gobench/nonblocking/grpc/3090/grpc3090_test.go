package grpc3090

import (
	"sync"
	"testing"
	"time"
	callstack "toolkit/pkg/callstack"
)

type resolver_ClientConn interface {
	UpdateState()
}

type resolver_Resolver struct {
	CC resolver_ClientConn
}

func (r *resolver_Resolver) Build(cc resolver_ClientConn) Resolver {
	defer callstack.Trace(158913789953)()
	r.CC = cc
	r.UpdateState()
	return r
}

func (r *resolver_Resolver) ResolveNow() {
}

func (r *resolver_Resolver) UpdateState() {
	defer callstack.Trace(158913789954)()
	r.CC.UpdateState()
}

type Resolver interface {
	ResolveNow()
}

type ccResolverWrapper struct {
	cc       *ClientConn
	resolver Resolver
	mu       sync.Mutex
}

func (ccr *ccResolverWrapper) resolveNow() {
	defer callstack.Trace(158913789955)()
	ccr.mu.Lock()
	ccr.resolver.ResolveNow()
	ccr.mu.Unlock()
}

func (ccr *ccResolverWrapper) poll() {
	defer callstack.Trace(158913789956)()
	ccr.mu.Lock()
	defer ccr.mu.Unlock()
	go func() {
		defer callstack.Trace(158913789957)()
		ccr.resolveNow()
	}()
}

func (ccr *ccResolverWrapper) UpdateState() {
	defer callstack.Trace(158913789958)()
	ccr.poll()
}

func newCCResolverWrapper(cc *ClientConn) {
	defer callstack.Trace(158913789959)()
	rb := cc.dopts.resolverBuilder
	ccr := &ccResolverWrapper{}
	ccr.resolver = rb.Build(ccr)
}

type Builder interface {
	Build(cc resolver_ClientConn) Resolver
}

type dialOptions struct {
	resolverBuilder Builder
}

type ClientConn struct {
	dopts dialOptions
}

func DialContext() {
	defer callstack.Trace(158913789960)()
	cc := &ClientConn{
		dopts: dialOptions{},
	}
	if cc.dopts.resolverBuilder == nil {
		cc.dopts.resolverBuilder = &resolver_Resolver{}
	}
	newCCResolverWrapper(cc)
}
func Dial() {
	defer callstack.Trace(158913789961)()
	DialContext()
}

func TestGrpc3090(t *testing.T) {
	defer callstack.Trace(158913789962)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(158913789963)()
		defer wg.Done()
		Dial()
		time.Sleep(5 * time.Millisecond)
	}()
	wg.Wait()
}
func TestGrpc3090_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(158913789962)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(158913789963)()
		defer wg.Done()
		Dial()
		time.Sleep(5 * time.Millisecond)
	}()
	wg.Wait()
}
