package goroutine

import (
	"bytes"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestSnapshotEdgesCountsAndSorts(t *testing.T) {
	gt := NewGoroutineTracker()
	gt.EnterGoroutineWithParent(7, 5)
	gt.EnterGoroutineWithParent(5, 0)
	gt.EnterGoroutineWithParent(5, 0)
	gt.EnterGoroutineWithParent(4, 9)
	gt.EnterGoroutineWithParent(5, 5)
	gt.EnterGoroutineWithParent(0, 0)

	want := []goroutineEdge{
		{ParentGid: 0, ChildGid: 5, Count: 2},
		{ParentGid: 5, ChildGid: 7, Count: 1},
		{ParentGid: 9, ChildGid: 4, Count: 1},
	}
	if got := gt.snapshotEdges(); !reflect.DeepEqual(got, want) {
		t.Fatalf("snapshotEdges() = %#v, want %#v", got, want)
	}
}

func TestWriteGoroutineEdges(t *testing.T) {
	edges := []goroutineEdge{
		{ParentGid: 0, ChildGid: 5, Count: 2},
		{ParentGid: 5, ChildGid: 7, Count: 1},
	}

	var output bytes.Buffer
	if err := writeGoroutineEdges(&output, edges); err != nil {
		t.Fatalf("writeGoroutineEdges() error = %v", err)
	}

	want := "[GORT_EDGE] {\"parent\":0,\"child\":5,\"count\":2}\n" +
		"[GORT_EDGE] {\"parent\":5,\"child\":7,\"count\":1}\n"
	if got := output.String(); got != want {
		t.Fatalf("writeGoroutineEdges() = %q, want %q", got, want)
	}
}

func TestPrintGoroutinePairsWritesEdgesWithoutPairs(t *testing.T) {
	oldTracker := tracker
	oldSkipRecord := skipRecord
	oldStderr := os.Stderr
	defer func() {
		tracker = oldTracker
		skipRecord = oldSkipRecord
		os.Stderr = oldStderr
	}()

	tracker = NewGoroutineTracker()
	tracker.EnterGoroutineWithParent(10, 0)
	skipRecord = false

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stderr = writer
	PrintGoroutinePairs()
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}
	os.Stderr = oldStderr

	output, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("reader.Close() error = %v", err)
	}

	want := `[GORT_EDGE] {"parent":0,"child":10,"count":1}`
	if !strings.Contains(string(output), want) {
		t.Fatalf("PrintGoroutinePairs() output = %q, want line %q", output, want)
	}
}
