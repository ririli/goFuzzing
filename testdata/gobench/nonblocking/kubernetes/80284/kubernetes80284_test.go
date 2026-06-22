package kubernetes80284

import (
	"sync"
	"testing"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

type Dialer struct{}

func (d *Dialer) CloseAll() {}

func NewDialer() *Dialer {
	return &Dialer{}
}

type Authenticator struct {
	onRotate func()
}

func (a *Authenticator) UpdateTransportConfig() {
	d := NewDialer()
	a.onRotate = d.CloseAll
}

func newAuthenticator() *Authenticator {
	return &Authenticator{}
}

func TestKubernetes80284(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(2)
	a := newAuthenticator()
	for i := 0; i < 2; i++ {
		go func() {
			goroutine.Enter(274877906945)
			defer goroutine.Exit(274877906945)
			func() {
				defer wg.Done()
				a.UpdateTransportConfig()
			}()
		}()
	}
	wg.Wait()
}
func TestKubernetes80284_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	wg.Add(2)
	a := newAuthenticator()
	for i := 0; i < 2; i++ {
		go func() {
			goroutine.Enter(274877906945)
			defer goroutine.Exit(274877906945)
			func() {
				defer wg.Done()
				a.UpdateTransportConfig()
			}()
		}()
	}
	wg.Wait()
}
