/*
 * Project: cockroach
 * Issue or PR  : https://github.com/cockroachdb/cockroach/pull/7504
 * Buggy version: bc963b438cdc3e0ad058a5282358e5aee0595e17
 * fix commit-id: cab761b9f5ee5dee1448bc5d6b1d9f5a0ff0bad5
 * Flaky: 1/100
 * Description: There are locking leaseState, tableNameCache in Release(), but
 * tableNameCache,LeaseState in AcquireByName.  It is AB and BA deadlock.
 */
package cockroach7504

import (
	sched "sched"
	"sync"
	"testing"
)

const tableSize = 1

func MakeCacheKey(lease *LeaseState) int {
	return lease.id
}

type LeaseState struct {
	mu sync.Mutex // LockA
	id int
}
type LeaseSet struct {
	data []*LeaseState
}

func (l *LeaseSet) insert(s *LeaseState) {
	s.id = len(l.data)
	l.data = append(l.data, s)
}
func (l *LeaseSet) find(id int) *LeaseState {
	return l.data[id]
}
func (l *LeaseSet) remove(s *LeaseState) {
	for i := 0; i < len(l.data); i++ {
		if s == l.data[i] {
			l.data = append(l.data[:i], l.data[i+1:]...)
			break
		}
	}
}

type tableState struct {
	tableNameCache *tableNameCache
	mu             sync.Mutex
	active         *LeaseSet
}

func (t *tableState) release(lease *LeaseState) {
	sched.InstMutexBF(227633266689, &t.mu)
	t.mu.Lock()
	sched.InstMutexAF(227633266689, &t.mu)
	defer func() {
		sched.InstMutexBF(227633266690, &t.mu)
		t.mu.Unlock()
		sched.InstMutexAF(227633266690, &t.mu)
	}()

	s := t.active.find(MakeCacheKey(lease))
	sched.InstMutexBF(227633266691,
		// LockA acquire
		&s.mu)
	s.mu.Lock()
	sched.InstMutexAF(227633266691, &s.mu)
	defer func() {
		sched.InstMutexBF(227633266692, &s.mu)
		s.mu.Unlock()
		sched. // LockA release
			InstMutexAF(227633266692, &s.mu)
	}()

	t.removeLease(s)
}
func (t *tableState) removeLease(lease *LeaseState) {
	t.active.remove(lease)
	t.tableNameCache.remove(lease) // LockA acquire/release
}

type tableNameCache struct {
	mu     sync.Mutex // LockB
	tables map[int]*LeaseState
}

func (c *tableNameCache) get(id int) {
	sched.InstMutexBF(
		// LockA acquire
		227633266693, &c.mu)
	c.mu.Lock()
	sched.InstMutexAF(227633266693, &c.mu)
	defer func() {
		sched.InstMutexBF(227633266694, &c.mu)
		c.mu.Unlock()
		sched.InstMutexAF(227633266694, &c.mu)
	}()
	lease, ok := c.tables[id]
	if !ok {
		return
	}
	if lease == nil {
		panic("nil lease in name cache")
	}
	sched.
		//+time.Sleep(time.Second)
		InstMutexBF(227633266695, &lease.mu)
	lease.mu.Lock()
	sched. // LockB acquire
		InstMutexAF(227633266695, &lease.mu)
	defer func() {
		sched.InstMutexBF(227633266696, &lease.mu)
		lease.mu.Unlock()
		sched.
			// LockB release
			// LockA release
			InstMutexAF(227633266696, &lease.mu)
	}()

}

func (c *tableNameCache) remove(lease *LeaseState) {
	sched.InstMutexBF(
		// LockA acquire
		227633266697, &c.mu)
	c.mu.Lock()
	sched.InstMutexAF(227633266697, &c.mu)
	defer func() {
		sched.InstMutexBF(227633266698, &c.mu)
		c.mu.Unlock()
		sched.InstMutexAF(227633266698, &c.mu)
	}()
	key := MakeCacheKey(lease)
	existing, ok := c.tables[key]
	if !ok {
		return
	}
	if existing == lease {
		delete(c.tables, key)
	}
	// LockA release
}

type LeaseManager struct {
	_          [64]byte
	mu         sync.Mutex
	tableNames *tableNameCache
	tables     map[int]*tableState
}

func (m *LeaseManager) AcquireByName(id int) {
	m.tableNames.get(id)
}

func (m *LeaseManager) findTableState(lease *LeaseState) *tableState {
	existing, ok := m.tables[0]
	if !ok {
		return nil
	}
	return existing
}

func (m *LeaseManager) Release(lease *LeaseState) {
	t := m.findTableState(lease)
	t.release(lease)
}
func NewLeaseManager(tname *tableNameCache, ts *tableState) *LeaseManager {
	mgr := &LeaseManager{
		tableNames: tname,
		tables:     make(map[int]*tableState),
	}
	mgr.tables[0] = ts
	return mgr
}
func NewLeaseSet(n int) *LeaseSet {
	lset := &LeaseSet{}
	for i := 0; i < n; i++ {
		lease := new(LeaseState)
		lset.data = append(lset.data, lease)
	}
	return lset
}

func TestCockroach7504(t *testing.T) {
	leaseNum := 2
	lset := NewLeaseSet(leaseNum)

	nc := &tableNameCache{
		tables: make(map[int]*LeaseState),
	}
	for i := 0; i < leaseNum; i++ {
		nc.tables[i] = lset.find(i)
	}

	ts := &tableState{
		tableNameCache: nc,
		active:         lset,
	}

	mgr := NewLeaseManager(nc, ts)
	var wg sync.WaitGroup

	wg.Add(2)
	// G1
	go func() {
		// lock AB
		mgr.AcquireByName(0)
		wg.Done()
	}()

	// G2
	go func() {
		// lock BA
		mgr.Release(lset.find(0))
		wg.Done()
	}()

	wg.Wait()
}
func TestCockroach7504_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	leaseNum := 2
	lset := NewLeaseSet(leaseNum)

	nc := &tableNameCache{
		tables: make(map[int]*LeaseState),
	}
	for i := 0; i < leaseNum; i++ {
		nc.tables[i] = lset.find(i)
	}

	ts := &tableState{
		tableNameCache: nc,
		active:         lset,
	}

	mgr := NewLeaseManager(nc, ts)
	var wg sync.WaitGroup

	wg.Add(2)

	go func() {

		mgr.AcquireByName(0)
		wg.Done()
	}()

	go func() {

		mgr.Release(lset.find(0))
		wg.Done()
	}()

	wg.Wait()
}
