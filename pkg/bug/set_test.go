package bug

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSetAddDeduplicatesAndTracksEvidence(t *testing.T) {
	set := NewSet()
	event := Event{
		Kind:      KindDataRace,
		Signature: "race-signature",
		Message:   "read a.go:1 <-> write b.go:2",
	}
	firstEvidence := Evidence{
		ExecutionID: 1,
		Mode:        "preexec",
		PairCovered: []Pair{{PreID: 10, NextID: 20}},
		Duration:    time.Second,
	}
	record, isNew := set.Add(event, firstEvidence)
	if !isNew || record.Count != 1 || record.AssociatedCount != 0 {
		t.Fatalf("first Add() = (%#v, %v), want a new count-1 record", record, isNew)
	}

	// Set 必须持有自己的 evidence 副本，而不是监视器传入的别名引用。
	firstEvidence.PairCovered[0].PreID = 999
	secondEvidence := Evidence{
		ExecutionID: 2,
		Mode:        "validate",
		PairCovered: []Pair{{PreID: 10, NextID: 20}},
		Associated:  true,
	}
	record, isNew = set.Add(event, secondEvidence)
	if isNew {
		t.Fatal("duplicate Add() reported a new event")
	}
	if record.Count != 2 || record.AssociatedCount != 1 {
		t.Fatalf("duplicate record counts = (%d, %d), want (2, 1)", record.Count, record.AssociatedCount)
	}
	if record.First.ExecutionID != 1 || record.Last.ExecutionID != 2 {
		t.Fatalf("evidence bounds = (%d, %d), want (1, 2)", record.First.ExecutionID, record.Last.ExecutionID)
	}
	if record.First.PairCovered[0].PreID != 10 {
		t.Fatal("stored evidence was mutated through the caller's slice")
	}
	if set.Len() != 1 || set.TriggeredLen() != 1 || !set.HasTriggered() {
		t.Fatalf("unexpected set counts: len=%d triggered=%d", set.Len(), set.TriggeredLen())
	}
}

func TestSetKeepsDifferentSignaturesAndKindsDistinct(t *testing.T) {
	set := NewSet()
	set.Add(Event{Kind: KindPanic, Signature: "a", Message: "panic a"}, Evidence{})
	set.Add(Event{Kind: KindPanic, Signature: "b", Message: "panic b"}, Evidence{})
	set.Add(Event{Kind: KindFatal, Signature: "a", Message: "fatal a"}, Evidence{})
	set.Add(Event{Kind: KindHangCandidate, Signature: "h", Message: "test timed out"}, Evidence{})

	if set.Len() != 4 {
		t.Fatalf("Len() = %d, want 4", set.Len())
	}
	if set.TriggeredLen() != 3 {
		t.Fatalf("TriggeredLen() = %d, want 3", set.TriggeredLen())
	}
}

func TestSetSnapshotAndSummaryAreStable(t *testing.T) {
	set := NewSet()
	set.Add(Event{Kind: KindPanic, Signature: "z", Message: "z panic"}, Evidence{})
	set.Add(Event{Kind: KindDataRace, Signature: "b", Message: "b race"}, Evidence{Associated: true})
	set.Add(Event{Kind: KindDataRace, Signature: "a", Message: "a race"}, Evidence{})

	snapshot := set.Snapshot()
	wantOrder := []string{"a", "b", "z"}
	for i, want := range wantOrder {
		if snapshot[i].Signature != want {
			t.Fatalf("Snapshot()[%d].Signature = %q, want %q", i, snapshot[i].Signature, want)
		}
	}

	want := strings.Join([]string{
		`kind=data_race signature=a count=1 associated=0 message="a race"`,
		`kind=data_race signature=b count=1 associated=1 message="b race"`,
		`kind=panic signature=z count=1 associated=0 message="z panic"`,
	}, "\n")
	if got := set.Summary(); got != want {
		t.Fatalf("Summary() =\n%s\nwant:\n%s", got, want)
	}
}

func TestSetIsConcurrencySafe(t *testing.T) {
	set := NewSet()
	event := Event{Kind: KindPanic, Signature: "same", Message: "boom"}
	const workers = 20
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func(id int) {
			defer wg.Done()
			set.Add(event, Evidence{ExecutionID: uint64(id)})
		}(i)
	}
	wg.Wait()

	records := set.Snapshot()
	if len(records) != 1 || records[0].Count != workers {
		t.Fatalf("concurrent Add() produced %#v, want one count-%d record", records, workers)
	}
}

func TestEventTriggered(t *testing.T) {
	for _, kind := range []Kind{KindDataRace, KindPanic, KindFatal} {
		if !(Event{Kind: kind}).Triggered() {
			t.Errorf("kind %q should trigger", kind)
		}
	}
	if (Event{Kind: KindHangCandidate}).Triggered() {
		t.Fatal("hang candidate should not trigger")
	}
}
