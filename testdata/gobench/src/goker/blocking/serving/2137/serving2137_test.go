package serving2137

import (
	"runtime"
	sched "sched"
	"sync"
	"testing"
)

type token struct{}

type request struct {
	lock     *sync.Mutex
	accepted chan bool
}

type Breaker struct {
	pendingRequests chan token
	activeRequests  chan token
}

func (b *Breaker) Maybe(thunk func()) bool {
	var t token
	select {
	default:
		// Pending request queue is full.  Report failure.
		return false
	case b.pendingRequests <- t:
		sched.
			// Pending request has capacity.
			// Wait for capacity in the active queue.
			InstChAF(936302870533, b.pendingRequests)
		sched.InstChBF(936302870530, b.activeRequests)
		b.activeRequests <- t
		sched.
			// Defer releasing capacity in the active and pending request queue.
			InstChAF(936302870530, b.activeRequests)

		defer func() { <-b.activeRequests; <-b.pendingRequests }()
		// Do the thing.
		thunk()
		// Report success
		return true
	}
}

func (b *Breaker) concurrentRequest() request {
	runtime.Gosched()

	r := request{lock: &sync.Mutex{}, accepted: make(chan bool, 1)}
	sched.InstMutexBF(936302870534, &r.lock)
	r.lock.Lock()
	sched.InstMutexAF(936302870534, &r.lock)
	var start sync.WaitGroup
	start.Add(1)
	go func() { // G2, G3
		start.Done()
		ok := b.Maybe(func() {
			sched.InstMutexBF(936302870535,
				// Will block on locked mutex.
				&r.lock)
			r.lock.Lock()
			sched.InstMutexAF(936302870535, &r.lock)
			sched.InstMutexBF(936302870536, &r.lock)
			r.lock.Unlock()
			sched.InstMutexAF(936302870536, &r.lock)
		})
		sched.InstChBF(936302870531, r.accepted)
		r.accepted <- ok
		sched.InstChAF(936302870531, r.accepted)
	}()
	start.Wait() // Ensure that the go func has had a chance to execute.
	return r
}

// Perform n requests against the breaker, returning mutexes for each
// request which succeeded, and a slice of bools for all requests.
func (b *Breaker) concurrentRequests(n int) []request {
	requests := make([]request, n)
	for i := range requests {
		requests[i] = b.concurrentRequest()
	}
	return requests
}

func NewBreaker(queueDepth, maxConcurrency int32) *Breaker {
	return &Breaker{
		pendingRequests: make(chan token, queueDepth+maxConcurrency),
		activeRequests:  make(chan token, maxConcurrency),
	}
}

func unlock(req request) {
	sched.InstMutexBF(936302870537,

		// Verify that function has completed
		&req.lock)
	req.lock.Unlock()
	sched.InstMutexAF(936302870537, &req.lock)

	ok := <-req.accepted
	sched.
		// Requeue for next usage
		InstChBF(936302870532, req.accepted)
	req.accepted <- ok
	sched.InstChAF(936302870532, req.accepted)
}

func unlockAll(requests []request) {
	for _, lc := range requests {
		unlock(lc)
	}
}

// G1                           G2                      G3
// b.concurrentRequests(2)
// b.concurrentRequest()
// r.lock.Lock()
//
//	start.Done()
//
// start.Wait()
// b.concurrentRequest()
// r.lock.Lock()
//
//	start.Done()
//
// start.Wait()
// unlockAll(locks)
// unlock(lc)
// req.lock.Unlock()
// ok := <-req.accepted
//
//	b.Maybe()
//	b.activeRequests <- t
//	thunk()
//	r.lock.Lock()
//	                        b.Maybe()
//	                        b.activeRequests <- t
//
// ----------------------------G1,G2,G3 deadlock-----------------------------
func TestServing2137(t *testing.T) {
	b := NewBreaker(1, 1)

	locks := b.concurrentRequests(2) // G1
	unlockAll(locks)
}
func TestServing2137_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	b := NewBreaker(1, 1)

	locks := b.concurrentRequests(2)
	unlockAll(locks)
}
