package kubernetes80284

import (
	"sync"
	"testing"
	callstack "toolkit/pkg/callstack"
)

type Dialer struct{}

func (d *Dialer) CloseAll() {}

func NewDialer() *Dialer {
	defer callstack.Trace(274877906945)()
	return &Dialer{}
}

type Authenticator struct {
	onRotate func()
}

func (a *Authenticator) UpdateTransportConfig() {
	defer callstack.Trace(274877906946)()
	d := NewDialer()
	a.onRotate = d.CloseAll
}

func newAuthenticator() *Authenticator {
	defer callstack.Trace(274877906947)()
	return &Authenticator{}
}

func TestKubernetes80284(t *testing.T) {
	defer callstack.Trace(274877906948)()
	var wg sync.WaitGroup
	wg.Add(2)
	a := newAuthenticator()
	for i := 0; i < 2; i++ {
		go func() {
			defer callstack.Trace(274877906949)()
			defer wg.Done()
			a.UpdateTransportConfig()
		}()
	}
	wg.Wait()
}
func TestKubernetes80284_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(274877906948)()
	var wg sync.WaitGroup
	wg.Add(2)
	a := newAuthenticator()
	for i := 0; i < 2; i++ {
		go func() {
			defer callstack.Trace(274877906949)()
			defer wg.Done()
			a.UpdateTransportConfig()
		}()
	}
	wg.Wait()
}
