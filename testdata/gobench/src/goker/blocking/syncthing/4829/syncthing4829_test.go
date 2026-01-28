package syncthing4829

import (
	sched "sched"
	"sync"
	"testing"
)

type Address int

type Mapping struct {
	mut sync.RWMutex

	extAddresses map[string]Address
}

func (m *Mapping) clearAddresses() {
	sched.InstMutexBF(
		// First locking
		833223655425, &m.mut)
	m.mut.Lock()
	sched.InstMutexAF(833223655425, &m.mut)
	var removed []Address
	for id, addr := range m.extAddresses {
		removed = append(removed, addr)
		delete(m.extAddresses, id)
	}
	if len(removed) > 0 {
		m.notify(nil, removed)
	}
	sched.InstMutexBF(833223655426, &m.mut)
	m.mut.Unlock()
	sched.InstMutexAF(833223655426, &m.mut)
}

func (m *Mapping) notify(added, remove []Address) {
	sched.InstMutexBF(
		// Second locking
		833223655427, &m.mut)
	m.mut.RLock()
	sched.InstMutexAF(833223655427, &m.mut)
	sched.InstMutexBF(833223655428, &m.mut)
	m.mut.RUnlock()
	sched.InstMutexAF(833223655428, &m.mut)
}

type Service struct {
	mut sync.RWMutex

	mappings []*Mapping
}

func (s *Service) NewMapping() *Mapping {
	mapping := &Mapping{
		extAddresses: make(map[string]Address),
	}
	sched.InstMutexBF(833223655429, &s.mut)
	s.mut.Lock()
	sched.InstMutexAF(833223655429, &s.mut)
	s.mappings = append(s.mappings, mapping)
	sched.InstMutexBF(833223655430, &s.mut)
	s.mut.Unlock()
	sched.InstMutexAF(833223655430, &s.mut)
	return mapping
}

func (s *Service) RemoveMapping(mapping *Mapping) {
	sched.InstMutexBF(833223655431, &s.mut)
	s.mut.Lock()
	sched.InstMutexAF(833223655431, &s.mut)
	defer func() {
		sched.InstMutexBF(833223655432, &s.mut)
		s.mut.Unlock()
		sched.InstMutexAF(833223655432, &s.mut)
	}()
	for _, existing := range s.mappings {
		if existing == mapping {
			mapping.clearAddresses()
		}
	}
}

func NewService() *Service {
	return &Service{}
}

func TestSyncthing4829(t *testing.T) {
	natSvc := NewService()
	m := natSvc.NewMapping()
	m.extAddresses["test"] = 0

	natSvc.RemoveMapping(m)
}
func TestSyncthing4829_1(t *testing.T) {
	sched.ParseInput()
	defer sched.Leakcheck(t)
	natSvc := NewService()
	m := natSvc.NewMapping()
	m.extAddresses["test"] = 0

	natSvc.RemoveMapping(m)
}
