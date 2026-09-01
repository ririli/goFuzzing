package bug

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Kind 标识产生事件的运行时预言机。
type Kind string

const (
	KindDataRace      Kind = "data_race"
	KindPanic         Kind = "panic"
	KindFatal         Kind = "fatal"
	KindHangCandidate Kind = "hang_candidate"
	// KindTestOrderPanic 疑似测试顺序依赖导致的 nil 接收者 panic：
	// GoPie 单独运行 _1 测试时，依赖其他测试初始化包级共享变量的测试
	// 会因变量未初始化而 nil 解引用。不计为可触发 bug，仅记录供人工甄别。
	KindTestOrderPanic Kind = "test_order_panic"
)

// Pair 记录一个成功的调度信号，不导入 feedback 包。
type Pair struct {
	PreID  uint64
	NextID uint64
}

// Event 是归一化后的运行时预言机事件。Signature 在动态地址、
// goroutine 编号、栈偏移和数据竞争访问顺序变化时保持稳定。
type Event struct {
	Kind      Kind
	Signature string
	Message   string
	Report    string
}

// Triggered 报告该事件是否足够强，可算作一个 bug。
// 超时和运行时死锁报告在重放能区分目标 bug 与预期的测试超时或
// 测试框架停顿之前，仍保持候选状态。
func (e Event) Triggered() bool {
	switch e.Kind {
	case KindDataRace, KindPanic, KindFatal:
		return true
	default:
		return false
	}
}

// Evidence 描述观察到事件的那次执行。
// Associated 表示同一次验证执行也覆盖了其输入对。
// 这是相关性证据，而非经重放确认的因果关系。
type Evidence struct {
	ExecutionID uint64
	Mode        string
	// Bin/Fn 记录产生该事件的测试二进制与测试函数，
	// 用于跨测试聚合报告（allpanic/alldatarace）中的来源定位。
	Bin string
	Fn  string
	// PairInput/PairCovered 为中性的并发对字段，
	// goroutine 与 function 两种粒度共用。
	PairInput   string
	OpInput     string
	PairCovered []Pair
	OpCovered   []Pair
	ExitError   string
	Duration    time.Duration
	Associated  bool
}

// Record 汇总对同一归一化事件的重复观测。
type Record struct {
	Event
	Count           int
	AssociatedCount int
	First           Evidence
	Last            Evidence
}

// Set 是不同归一化事件的并发安全集合。
type Set struct {
	mu      sync.RWMutex
	records map[string]*Record
}

func NewSet() *Set {
	return &Set{records: make(map[string]*Record)}
}

// Add 插入或更新一个事件。仅当某个 kind/signature 组合首次被观测到时，
// isNew 为 true。
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

// Len 返回不同事件的数量，包括挂起候选（hang candidate）。
func (s *Set) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.records)
}

// TriggeredLen 返回不同数据竞争、panic 和 fatal 事件的数量。
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

// Snapshot 按确定的 kind/signature 顺序返回深拷贝。
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

// Summary 为每个不同事件提供一行稳定、简洁的描述。
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
	evidence.PairCovered = append([]Pair(nil), evidence.PairCovered...)
	evidence.OpCovered = append([]Pair(nil), evidence.OpCovered...)
	return evidence
}
