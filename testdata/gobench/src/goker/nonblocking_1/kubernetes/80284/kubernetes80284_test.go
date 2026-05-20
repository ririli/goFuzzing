package kubernetes80284

import (
	"sync"
	"testing"
	callstack "toolkit/pkg/callstack"
)

type Dialer struct{}

func (d *Dialer) CloseAll() {}

func NewDialer() *Dialer {
	defer callstack.Trace(545460846593)()
	return &Dialer{}
}

type Authenticator struct {
	onRotate func()
}

func (a *Authenticator) UpdateTransportConfig() {
	defer callstack.Trace(545460846594)()
	d := NewDialer()
	a.onRotate = d.CloseAll
}

func newAuthenticator() *Authenticator {
	defer callstack.Trace(545460846595)()
	return &Authenticator{}
}

func TestKubernetes80284(t *testing.T) {
	defer callstack.Trace(545460846596)()
	var wg sync.WaitGroup
	wg.Add(2)
	a := newAuthenticator()
	for i := 0; i < 2; i++ {
		go func() {
			defer callstack.Trace(545460846597)()
			defer wg.Done()
			a.UpdateTransportConfig()
		}()
	}
	wg.Wait()
}
func TestKubernetes80284_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.Trace(545460846596)()
	var wg sync.WaitGroup
	wg.Add(2)
	a := newAuthenticator()
	for i := 0; i < 2; i++ {
		go func() {
			defer callstack.Trace(545460846597)()
			defer wg.Done()
			a.UpdateTransportConfig()
		}()
	}
	wg.Wait()
}
