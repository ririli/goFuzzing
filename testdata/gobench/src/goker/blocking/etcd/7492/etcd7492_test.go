/*
 * Project: etcd
 * Issue or PR  : https://github.com/etcd-io/etcd/pull/7492
 * Buggy version: 51939650057d602bb5ab090633138fffe36854dc
 * fix commit-id: 1b1fabef8ffec606909f01c3983300fff539f214
 * Flaky: 40/100
 */
package etcd7492

import (
	sched "sched"
	"sync"
	"testing"
	"time"
)

type TokenProvider interface {
	assign()
	enable()
	disable()
}

type simpleTokenTTLKeeper struct {
	tokens           map[string]time.Time
	addSimpleTokenCh chan struct{}
	stopCh           chan chan struct{}
	deleteTokenFunc  func(string)
}

type authStore struct {
	tokenProvider TokenProvider
}

func (as *authStore) Authenticate() {
	as.tokenProvider.assign()
}

func NewSimpleTokenTTLKeeper(deletefunc func(string)) *simpleTokenTTLKeeper {
	stk := &simpleTokenTTLKeeper{
		tokens:           make(map[string]time.Time),
		addSimpleTokenCh: make(chan struct{}, 1),
		stopCh:           make(chan chan struct{}),
		deleteTokenFunc:  deletefunc,
	}
	go stk.run() // G1
	return stk
}

func (tm *simpleTokenTTLKeeper) run() {
	tokenTicker := time.NewTicker(time.Nanosecond)
	defer tokenTicker.Stop()
	for {
		select {
		case <-tm.addSimpleTokenCh:
			sched.
				/// Make tm.tokens not empty is enough
				InstChAF(704374636552, tm.addSimpleTokenCh)

			tm.tokens["1"] = time.Now()
		case <-tokenTicker.C:
			sched.InstChAF(704374636553, tokenTicker.C)
			for t, _ := range tm.tokens {
				tm.deleteTokenFunc(t)
				delete(tm.tokens, t)
			}
		case waitCh := <-tm.stopCh:
			sched.InstChAF(704374636554, tm.stopCh)
			sched.InstChBF(704374636547, waitCh)
			waitCh <- struct{}{}
			sched.InstChAF(704374636547, waitCh)
			return
		}
	}
}

func (tm *simpleTokenTTLKeeper) addSimpleToken() {
	sched.InstChBF(704374636548, tm.addSimpleTokenCh)
	tm.addSimpleTokenCh <- struct{}{}
	sched.InstChAF(704374636548, tm.addSimpleTokenCh)
}

func (tm *simpleTokenTTLKeeper) stop() {
	waitCh := make(chan struct{})
	sched.InstChBF(704374636549, tm.stopCh)
	tm.stopCh <- waitCh
	sched.InstChAF(704374636549, tm.stopCh)
	sched.InstChBF(704374636550, waitCh)
	<-waitCh
	sched.InstChAF(704374636550, waitCh)
	close(tm.stopCh)
}

type tokenSimple struct {
	simpleTokenKeeper *simpleTokenTTLKeeper
	simpleTokensMu    sync.RWMutex
}

func (t *tokenSimple) assign() {
	t.assignSimpleTokenToUser()
}

func (t *tokenSimple) assignSimpleTokenToUser() {
	sched.InstMutexBF(704374636555, &t.simpleTokensMu)
	t.simpleTokensMu.Lock()
	sched.InstMutexAF(704374636555, &t.simpleTokensMu)
	t.simpleTokenKeeper.addSimpleToken()
	sched.InstMutexBF(704374636556, &t.simpleTokensMu)
	t.simpleTokensMu.Unlock()
	sched.InstMutexAF(704374636556, &t.simpleTokensMu)
}
func newDeleterFunc(t *tokenSimple) func(string) {
	return func(tk string) {
		sched.InstMutexBF(704374636557, &t.simpleTokensMu)
		t.simpleTokensMu.Lock()
		sched.InstMutexAF(704374636557, &t.simpleTokensMu)
		defer func() {
			sched.InstMutexBF(704374636558, &t.simpleTokensMu)
			t.simpleTokensMu.Unlock()
			sched.InstMutexAF(704374636558, &t.simpleTokensMu)
		}()
	}
}

func (t *tokenSimple) enable() {
	t.simpleTokenKeeper = NewSimpleTokenTTLKeeper(newDeleterFunc(t))
}

func (t *tokenSimple) disable() {
	if t.simpleTokenKeeper != nil {
		t.simpleTokenKeeper.stop()
		t.simpleTokenKeeper = nil
	}
	sched.InstMutexBF(704374636559, &t.simpleTokensMu)
	t.simpleTokensMu.Lock()
	sched.InstMutexAF(704374636559, &t.simpleTokensMu)
	sched.InstMutexBF(704374636560, &t.simpleTokensMu)
	t.simpleTokensMu.Unlock()
	sched.InstMutexAF(704374636560, &t.simpleTokensMu)
}

func newTokenProviderSimple() *tokenSimple {
	return &tokenSimple{}
}

func setupAuthStore() (store *authStore, teardownfunc func()) {
	as := &authStore{
		tokenProvider: newTokenProviderSimple(),
	}
	as.tokenProvider.enable()
	tearDown := func() {
		as.tokenProvider.disable()
	}
	return as, tearDown
}

// /
// /	G1										G2
// /											stk.run()
// /	ts.assignSimpleTokenToUser()
// /	t.simpleTokensMu.Lock()
// /	t.simpleTokenKeeper.addSimpleToken()
// /	tm.addSimpleTokenCh <- true
// /											<-tm.addSimpleTokenCh
// /	t.simpleTokensMu.Unlock()
// /	ts.assignSimpleTokenToUser()
// /	...										...
// /	t.simpleTokensMu.Lock()
// /											<-tokenTicker.C
// /	tm.addSimpleTokenCh <- true
// /											tm.deleteTokenFunc()
// /											t.simpleTokensMu.Lock()
// /------------------------------------G1,G2 deadlock---------------------------------------------
// /
func TestEtcd7492(t *testing.T) {
	as, tearDown := setupAuthStore()
	defer tearDown()
	var wg sync.WaitGroup
	wg.Add(3)
	for i := 0; i < 3; i++ {
		go func() { // G2
			defer wg.Done()
			as.Authenticate()
		}()
	}
	wg.Wait()
}
func TestEtcd7492_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	as, tearDown := setupAuthStore()
	defer tearDown()
	var wg sync.WaitGroup
	wg.Add(3)
	for i := 0; i < 3; i++ {
		go func() {
			defer wg.Done()
			as.Authenticate()
		}()
	}
	wg.Wait()
}
