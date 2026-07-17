package fuzzer

import (
	"errors"
	"testing"
	"time"

	"toolkit/pkg/bug"
	"toolkit/pkg/feedback"
)

const monitorRaceReport = `==================
WARNING: DATA RACE
Read at 0x00c000012100 by goroutine 9:
  example.com/project.reader()
      /work/project/reader.go:21 +0x31

Previous write at 0x00c000012100 by goroutine 8:
  example.com/project.writer()
      /work/project/writer.go:34 +0xa4
==================
`

func TestAnalyzeRunValidateCoveredRaceIsAssociated(t *testing.T) {
	bugs := bug.NewSet()
	ctx := RunContext{
		In: Input{gortPair: &feedback.InputGortPair{TryPair: []*feedback.GortPairInfo{
			{Gid1: 20, Gid2: 10},
		}}},
		Out: Output{
			O:     "{COVERED} {10, 20}\n",
			Trace: monitorRaceReport,
			Err:   errors.New("exit status 66"),
			Time:  125 * time.Millisecond,
		},
	}

	analysis := analyzeRun(ctx, 42, bugs)
	if !analysis.triggered || !analysis.newOracleFinding {
		t.Fatalf("analysis = triggered:%t new:%t, want both true", analysis.triggered, analysis.newOracleFinding)
	}
	if len(analysis.gortSignals) != 1 {
		t.Fatalf("gort signals = %d, want 1", len(analysis.gortSignals))
	}
	records := bugs.Snapshot()
	if len(records) != 1 {
		t.Fatalf("bug records = %d, want 1", len(records))
	}
	evidence := records[0].Last
	if evidence.ExecutionID != 42 || evidence.Mode != "validate" {
		t.Fatalf("evidence execution/mode = %d/%q, want 42/validate", evidence.ExecutionID, evidence.Mode)
	}
	if !evidence.Associated {
		t.Fatal("matching COVERED signal should associate the race with the validation execution")
	}
	if evidence.GortInput != "(20,10)" {
		t.Fatalf("gort input = %q, want %q", evidence.GortInput, "(20,10)")
	}
	if len(evidence.GortCovered) != 1 || evidence.GortCovered[0] != (bug.Pair{PreID: 10, NextID: 20}) {
		t.Fatalf("covered pairs = %#v, want [{10 20}]", evidence.GortCovered)
	}
	if evidence.ExitError != "exit status 66" || evidence.Duration != 125*time.Millisecond {
		t.Fatalf("exit evidence = %q/%s", evidence.ExitError, evidence.Duration)
	}
}

func TestAnalyzeRunPreexecRaceTriggersWithoutAssociation(t *testing.T) {
	bugs := bug.NewSet()
	analysis := analyzeRun(RunContext{Out: Output{Trace: monitorRaceReport}}, 1, bugs)

	if !analysis.triggered {
		t.Fatal("pre-execution race should still trigger the bug oracle")
	}
	record := bugs.Snapshot()[0]
	if record.Last.Mode != "preexec" || record.Last.Associated {
		t.Fatalf("preexec evidence = mode:%q associated:%t", record.Last.Mode, record.Last.Associated)
	}
}

func TestAnalyzeRunRaceWithoutCoveredSignalIsNotAssociated(t *testing.T) {
	bugs := bug.NewSet()
	ctx := RunContext{
		In: Input{gortPair: &feedback.InputGortPair{TryPair: []*feedback.GortPairInfo{
			{Gid1: 10, Gid2: 20},
		}}},
		Out: Output{Trace: monitorRaceReport},
	}

	analysis := analyzeRun(ctx, 2, bugs)
	if !analysis.triggered {
		t.Fatal("race should trigger independently of scheduling coverage")
	}
	if bugs.Snapshot()[0].Last.Associated {
		t.Fatal("race without a matching successful COVERED signal must not be associated")
	}
}

func TestAnalyzeRunDuplicateBugIsNotNewProgress(t *testing.T) {
	bugs := bug.NewSet()
	ctx := RunContext{Out: Output{Trace: monitorRaceReport}}
	first := analyzeRun(ctx, 1, bugs)
	second := analyzeRun(ctx, 2, bugs)

	if !first.newOracleFinding || second.newOracleFinding {
		t.Fatalf("new finding flags = first:%t second:%t, want true/false", first.newOracleFinding, second.newOracleFinding)
	}
	records := bugs.Snapshot()
	if len(records) != 1 || records[0].Count != 2 || records[0].Last.ExecutionID != 2 {
		t.Fatalf("deduplicated records = %#v", records)
	}
}

func TestMatchesCoveredInputPairDirection(t *testing.T) {
	gortInput := Input{gortPair: &feedback.InputGortPair{TryPair: []*feedback.GortPairInfo{
		{Gid1: 20, Gid2: 10},
	}}}
	if !matchesCoveredInput(gortInput, []*feedback.CoverageSignal{{
		PreID: 10, NextID: 20, Success: true, Kind: feedback.SignalGortCovered,
	}}, nil) {
		t.Fatal("goroutine pair matching should be direction-insensitive")
	}

	opInput := Input{
		gortPair: &feedback.InputGortPair{},
		tryOpPair: &feedback.InputOpPair{TryPair: []*feedback.OpPair{{
			Op1: &feedback.OpInfo{OpId: 11},
			Op2: &feedback.OpInfo{OpId: 22},
		}}},
	}
	reversed := []*feedback.CoverageSignal{{
		PreID: 22, NextID: 11, Success: true, Kind: feedback.SignalOpCovered,
	}}
	if matchesCoveredInput(opInput, nil, reversed) {
		t.Fatal("operation pair matching should preserve scheduling direction")
	}
	forward := []*feedback.CoverageSignal{{
		PreID: 11, NextID: 22, Success: true, Kind: feedback.SignalOpCovered,
	}}
	if !matchesCoveredInput(opInput, nil, forward) {
		t.Fatal("matching operation scheduling direction should associate the execution")
	}
}

func TestHangCandidateDoesNotFailOrStopSingleCrash(t *testing.T) {
	bugs := bug.NewSet()
	analysis := analyzeRun(RunContext{Out: Output{
		Trace: "panic: test timed out after 30s\n\ngoroutine 1 [running]:\nexample.com/project.TestWork()\n\t/work/project/work_test.go:12 +0x20\n",
	}}, 3, bugs)

	if analysis.triggered {
		t.Fatal("hang candidate must not be classified as BugTriggered")
	}
	if !analysis.newOracleFinding || bugs.Len() != 1 {
		t.Fatalf("hang candidate should be retained as oracle progress: new=%t len=%d", analysis.newOracleFinding, bugs.Len())
	}
	if shouldStopAfterRun(true, analysis) {
		t.Fatal("SingleCrash must not stop for a hang candidate")
	}
	failed, detail := monitorResult(bugs)
	if failed || len(detail) < 2 || detail[0] != "PASS" {
		t.Fatalf("monitor result = %t, %#v; want PASS with two detail fields", failed, detail)
	}
}

func TestNilMonitorLoggerDoesNotBlockFindingLogging(t *testing.T) {
	bugs := bug.NewSet()
	analysis := analyzeRun(RunContext{Out: Output{Trace: monitorRaceReport}}, 5, bugs)
	done := make(chan struct{})
	go func() {
		logBugFindings(nil, 1, 5, analysis.findings)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("nil monitor logger blocked")
	}

	logCh := make(chan string, 1)
	sendMonitorLog(logCh, "finding")
	if got := <-logCh; got != "finding" {
		t.Fatalf("logged message = %q, want finding", got)
	}
}

func TestMonitorResultAlwaysHasCallerSafeDetail(t *testing.T) {
	failed, detail := monitorResult(bug.NewSet())
	if failed || len(detail) < 2 || detail[0] != "PASS" {
		t.Fatalf("empty result = %t, %#v", failed, detail)
	}

	bugs := bug.NewSet()
	analysis := analyzeRun(RunContext{Out: Output{Trace: monitorRaceReport}}, 4, bugs)
	if !shouldStopAfterRun(true, analysis) {
		t.Fatal("SingleCrash should stop after a race/panic/fatal event")
	}
	failed, detail = monitorResult(bugs)
	if !failed || len(detail) < 2 || detail[0] != "FAIL" || detail[1] == "" {
		t.Fatalf("triggered result = %t, %#v", failed, detail)
	}
}
