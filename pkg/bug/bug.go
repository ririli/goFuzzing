package bug

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Kind identifies the runtime oracle that produced an event.
type Kind string

const (
	KindDataRace      Kind = "data_race"
	KindPanic         Kind = "panic"
	KindFatal         Kind = "fatal"
	KindHangCandidate Kind = "hang_candidate"
)

// Pair records one successful scheduling signal without importing feedback.
type Pair struct {
	PreID  uint64
	NextID uint64
}

// Event is a normalized runtime oracle event. Signature is stable across
// dynamic addresses, goroutine numbers, stack offsets, and race access order.
type Event struct {
	Kind      Kind
	Signature string
	Message   string
	Report    string
}

// Triggered reports whether the event is strong enough to count as a bug.
// Timeouts and runtime deadlock reports remain candidates until replay can
// distinguish a target bug from an expected test timeout or harness stall.
func (e Event) Triggered() bool {
	switch e.Kind {
	case KindDataRace, KindPanic, KindFatal:
		return true
	default:
		return false
	}
}

// Evidence describes the execution in which an event was observed.
// Associated means that the same validation execution also covered its input
// pair. It is correlation evidence, not replay-confirmed causality.
type Evidence struct {
	ExecutionID uint64
	Mode        string
	GortInput   string
	OpInput     string
	GortCovered []Pair
	OpCovered   []Pair
	ExitError   string
	Duration    time.Duration
	Associated  bool
}

// Record aggregates repeated observations of one normalized event.
type Record struct {
	Event
	Count           int
	AssociatedCount int
	First           Evidence
	Last            Evidence
}

// Set is a concurrency-safe collection of distinct normalized events.
type Set struct {
	mu      sync.RWMutex
	records map[string]*Record
}

func NewSet() *Set {
	return &Set{records: make(map[string]*Record)}
}

// Add inserts or updates an event. isNew is true only for the first
// observation of a kind/signature pair.
func (s *Set) Add(event Event, evidence Evidence) (Record, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.records == nil {
		s.records = make(map[string]*Record)
	}
	evidence = cloneEvidence(evidence)
	key := string(event.Kind) + "\x00" + event.Signature
	if existing, ok := s.records[key]; ok {
		existing.Count++
		if evidence.Associated {
			existing.AssociatedCount++
		}
		existing.Last = evidence
		return cloneRecord(*existing), false
	}

	associatedCount := 0
	if evidence.Associated {
		associatedCount = 1
	}
	stored := &Record{
		Event:           event,
		Count:           1,
		AssociatedCount: associatedCount,
		First:           evidence,
		Last:            cloneEvidence(evidence),
	}
	s.records[key] = stored
	return cloneRecord(*stored), true
}

// Len returns the number of distinct events, including hang candidates.
func (s *Set) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}

// TriggeredLen returns the number of distinct race, panic, and fatal events.
func (s *Set) TriggeredLen() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	n := 0
	for _, record := range s.records {
		if record.Event.Triggered() {
			n++
		}
	}
	return n
}

func (s *Set) HasTriggered() bool {
	return s.TriggeredLen() != 0
}

// Snapshot returns a deep copy in deterministic kind/signature order.
func (s *Set) Snapshot() []Record {
	s.mu.RLock()
	defer s.mu.RUnlock()

	records := make([]Record, 0, len(s.records))
	for _, record := range s.records {
		records = append(records, cloneRecord(*record))
	}
	sort.Slice(records, func(i, j int) bool {
		if records[i].Kind != records[j].Kind {
			return records[i].Kind < records[j].Kind
		}
		return records[i].Signature < records[j].Signature
	})
	return records
}

// Summary provides one stable, concise line per distinct event.
func (s *Set) Summary() string {
	records := s.Snapshot()
	lines := make([]string, 0, len(records))
	for _, record := range records {
		lines = append(lines, fmt.Sprintf(
			"kind=%s signature=%s count=%d associated=%d message=%q",
			record.Kind,
			record.Signature,
			record.Count,
			record.AssociatedCount,
			record.Message,
		))
	}
	return strings.Join(lines, "\n")
}

func cloneRecord(record Record) Record {
	record.First = cloneEvidence(record.First)
	record.Last = cloneEvidence(record.Last)
	return record
}

func cloneEvidence(evidence Evidence) Evidence {
	evidence.GortCovered = append([]Pair(nil), evidence.GortCovered...)
	evidence.OpCovered = append([]Pair(nil), evidence.OpCovered...)
	return evidence
}
