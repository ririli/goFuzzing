package kubernetes80284

import (
	"sync"
	"testing"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/operation"
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
	sched.InstWgBF(18227557805101940738)
	wg.Add(2)
	sched.InstWgAF(18227557805101940738, &wg, "add")
	a := newAuthenticator()
	for i := 0; i < 2; i++ {
		go func(_parentGid uint64) {
			goroutine.Enter(18227557805101940737, _parentGid)
			defer goroutine.Exit(18227557805101940737)
			func() {
				defer func() {
					sched.InstWgBF(18227557805101940739)
					wg.Done()
					sched.InstWgAF(18227557805101940739, &wg, "done")
				}()
				a.UpdateTransportConfig()
			}()
		}(goroutine.CurrentGid())
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
	sched.InstWgBF(18227557805101940738)
	wg.Add(2)
	sched.InstWgAF(18227557805101940738, &wg, "add")
	a := newAuthenticator()
	for i := 0; i < 2; i++ {
		go func(_parentGid uint64) {
			goroutine.Enter(18227557805101940737, _parentGid)
			defer goroutine.Exit(18227557805101940737)
			func() {
				defer func() {
					sched.InstWgBF(18227557805101940739)
					wg.Done()
					sched.InstWgAF(18227557805101940739, &wg, "done")
				}()
				a.UpdateTransportConfig()
			}()
		}(goroutine.CurrentGid())
	}
	wg.Wait()
}
