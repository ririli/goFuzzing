package bug

import (
	"strings"
	"testing"
)

const raceReport = `==================
WARNING: DATA RACE
Read at 0x00c00010a570 by goroutine 9:
  example.com/project.readValue()
      /work/project/state.go:19 +0x31

Previous write at 0x00c00010a570 by goroutine 8:
  example.com/project.writeValue()
      /work/project/update.go:57 +0xa4

Goroutine 9 (running) created at:
  example.com/project.startReader()
      /work/project/main.go:80 +0x17a
==================`

func TestParseDataRace(t *testing.T) {
	events := Parse("", raceReport)
	if len(events) != 1 {
		t.Fatalf("Parse() returned %d events, want 1: %#v", len(events), events)
	}
	event := events[0]
	if event.Kind != KindDataRace {
		t.Fatalf("Kind = %q, want %q", event.Kind, KindDataRace)
	}
	if !event.Triggered() {
		t.Fatal("data race must trigger the bug oracle")
	}
	if len(event.Signature) != 64 {
		t.Errorf("signature length = %d, want 64", len(event.Signature))
	}
	for _, want := range []string{
		"read example.com/project.readValue@state.go:19",
		"write example.com/project.writeValue@update.go:57",
	} {
		if !strings.Contains(event.Message, want) {
			t.Errorf("Message %q does not contain %q", event.Message, want)
		}
	}
	if !strings.Contains(event.Report, "0x00c00010a570") {
		t.Error("Report should retain the original diagnostic")
	}
}

func TestRaceSignatureIgnoresOrderAndDynamicValues(t *testing.T) {
	swapped := `WARNING: DATA RACE
Write at 0x000000000001 by goroutine 123:
  example.com/project.writeValue(0xc000999999)
      /another/root/update.go:57 +0xffff

Previous read at 0x000000000001 by goroutine 456:
  example.com/project.readValue(0xc000888888)
      /another/root/state.go:19 +0x1
==================`

	first := Parse("", raceReport)
	second := Parse(swapped, "")
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("unexpected event counts: first=%d second=%d", len(first), len(second))
	}
	if first[0].Signature != second[0].Signature {
		t.Fatalf("signatures differ after access-order/dynamic-value changes:\n%s\n%s", first[0].Signature, second[0].Signature)
	}
}

func TestRaceUsesFirstApplicationFrame(t *testing.T) {
	input := `WARNING: DATA RACE
Read at 0x01 by goroutine 1:
  runtime.mapaccess1_faststr()
      /usr/local/go/src/runtime/map_faststr.go:13 +0x0
  example.com/project.readValue()
      /work/project/state.go:19 +0x31

Previous write at 0x01 by goroutine 2:
  sync.(*Map).Store()
      /usr/local/go/src/sync/map.go:158 +0x10
  example.com/project.writeValue()
      /work/project/update.go:57 +0xa4
==================`

	events := Parse("", input)
	if len(events) != 1 {
		t.Fatalf("Parse() returned %d events, want 1", len(events))
	}
	if strings.Contains(events[0].Message, "runtime.") || strings.Contains(events[0].Message, "sync.") {
		t.Fatalf("Message contains a non-application frame: %q", events[0].Message)
	}
	if !strings.Contains(events[0].Message, "state.go:19") || !strings.Contains(events[0].Message, "update.go:57") {
		t.Fatalf("Message does not identify both application accesses: %q", events[0].Message)
	}
}

func TestParseMultipleDistinctRaceBlocks(t *testing.T) {
	second := strings.ReplaceAll(raceReport, "state.go:19", "other.go:91")
	events := Parse(raceReport+"\n"+second, "")
	if len(events) != 2 {
		t.Fatalf("Parse() returned %d events, want 2", len(events))
	}
	if events[0].Signature == events[1].Signature {
		t.Fatal("distinct access locations produced the same signature")
	}
}

func TestParseDeduplicatesSameRaceAcrossStreams(t *testing.T) {
	events := Parse(raceReport, raceReport)
	if len(events) != 1 {
		t.Fatalf("Parse() returned %d events, want one deduplicated event", len(events))
	}
}

func TestParsePanicRequiresDiagnosticAtLineStart(t *testing.T) {
	input := "ordinary log says panic: not a runtime diagnostic\nstatus=panic: still ordinary"
	if events := Parse(input, ""); len(events) != 0 {
		t.Fatalf("Parse() returned false-positive events: %#v", events)
	}
}

func TestParsePanicWithApplicationFrame(t *testing.T) {
	first := `panic: send on closed channel

goroutine 20 [running]:
runtime.gopanic({0x111, 0x222})
	/usr/local/go/src/runtime/panic.go:890 +0x262
example.com/project.(*worker).send(0xc000111111)
	/work/project/worker.go:42 +0x67`
	second := strings.NewReplacer(
		"goroutine 20", "goroutine 987",
		"0x111", "0xaaaa",
		"0x222", "0xbbbb",
		"0xc000111111", "0xc000999999",
		"+0x67", "+0xff",
	).Replace(first)

	eventsA := Parse(first, "")
	eventsB := Parse("", second)
	if len(eventsA) != 1 || len(eventsB) != 1 {
		t.Fatalf("unexpected event counts: first=%d second=%d", len(eventsA), len(eventsB))
	}
	if eventsA[0].Kind != KindPanic || eventsA[0].Message != "send on closed channel" {
		t.Fatalf("unexpected panic event: %#v", eventsA[0])
	}
	if eventsA[0].Signature != eventsB[0].Signature {
		t.Fatal("dynamic panic stack values changed the signature")
	}
}

func TestParseRecoveredPanicDeduplicatesRepeatedHeader(t *testing.T) {
	input := `panic: send on closed channel [recovered]
	panic: send on closed channel

goroutine 19 [running]:
testing.tRunner.func1.2({0x5c1620, 0x628d90})
	/usr/local/go/src/testing/testing.go:1396 +0x372
runtime.gopanic({0x5c1620, 0x628d90})
	/usr/local/go/src/runtime/panic.go:890 +0x262
example.com/project.(*manager).update()
	/work/project/manager.go:26 +0x10`

	events := Parse("", input)
	if len(events) != 1 {
		t.Fatalf("Parse() returned %d events, want 1: %#v", len(events), events)
	}
	if events[0].Kind != KindPanic || events[0].Message != "send on closed channel" {
		t.Fatalf("unexpected recovered panic event: %#v", events[0])
	}
}

func TestParseRepanickedAnnotationMatchesPlainPanic(t *testing.T) {
	stack := `
goroutine 19 [running]:
example.com/project.run()
	/work/project/run.go:25 +0x10`
	repanicked := Parse("", "panic: outer [recovered, repanicked]"+stack)
	plain := Parse("", "panic: outer"+stack)
	if len(repanicked) != 1 || len(plain) != 1 {
		t.Fatalf("unexpected event counts: repanicked=%d plain=%d", len(repanicked), len(plain))
	}
	if repanicked[0].Signature != plain[0].Signature {
		t.Fatal("repanicked annotation changed the panic signature")
	}
}

func TestParseRecoveredSandwichPrefersFramedDuplicate(t *testing.T) {
	input := `panic: outer [recovered]
	panic: inner [recovered]
	panic: outer

goroutine 19 [running]:
example.com/project.run()
	/work/project/run.go:25 +0x10`

	events := Parse("", input)
	if len(events) != 2 {
		t.Fatalf("Parse() returned %d events, want outer and inner: %#v", len(events), events)
	}
	counts := make(map[string]int)
	for _, event := range events {
		counts[event.Message]++
	}
	if counts["outer"] != 1 || counts["inner"] != 1 {
		t.Fatalf("panic message counts = %#v, want one outer and one inner", counts)
	}
	for _, event := range events {
		if event.Message == "outer" && !strings.Contains(event.Report, "example.com/project.run") {
			t.Fatal("outer event retained the unframed recovered header")
		}
	}
}

func TestParseFatal(t *testing.T) {
	input := `fatal error: concurrent map writes

goroutine 8 [running]:
runtime.throw({0x1, 0x2})
	/usr/local/go/src/runtime/panic.go:1047 +0x5d
example.com/project.updateMap()
	/work/project/cache.go:71 +0x44`

	events := Parse("", input)
	if len(events) != 1 {
		t.Fatalf("Parse() returned %d events, want 1", len(events))
	}
	if events[0].Kind != KindFatal || !events[0].Triggered() {
		t.Fatalf("fatal event was not triggered: %#v", events[0])
	}
	if events[0].Message != "concurrent map writes" {
		t.Errorf("Message = %q, want concurrent map writes", events[0].Message)
	}
}

func TestPanicSignatureIgnoresCreatedByGoroutineID(t *testing.T) {
	makeReport := func(gid string) string {
		return `panic: worker failed

goroutine 20 [running]:
runtime.gopanic({0x1, 0x2})
	/usr/local/go/src/runtime/panic.go:890 +0x262
created by example.com/project.start in goroutine ` + gid + `
	/work/project/main.go:40 +0x10`
	}

	first := Parse("", makeReport("7"))
	second := Parse("", makeReport("999"))
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("unexpected event counts: first=%d second=%d", len(first), len(second))
	}
	if first[0].Signature != second[0].Signature {
		t.Fatal("created-by goroutine ID changed the panic signature")
	}
}

func TestPanicCreatedByMethodPreservesReceiverAndIgnoresGoroutineID(t *testing.T) {
	makeReport := func(gid string) string {
		return `panic: worker failed

goroutine 20 [running]:
runtime.gopanic({0x1, 0x2})
	/usr/local/go/src/runtime/panic.go:890 +0x262
created by example.com/project.(*Server).Start in goroutine ` + gid + `
	/work/project/server.go:40 +0x10`
	}

	first := Parse("", makeReport("7"))
	second := Parse("", makeReport("999"))
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("unexpected event counts: first=%d second=%d", len(first), len(second))
	}
	if first[0].Signature != second[0].Signature {
		t.Fatal("created-by goroutine ID changed the method panic signature")
	}

	wantSignature := signature("panic\nworker failed\nexample.com/project.(*Server).Start@server.go:40")
	if first[0].Signature != wantSignature {
		t.Fatalf("method receiver was lost from the application frame: got %s, want %s", first[0].Signature, wantSignature)
	}
}

func TestRuntimeBoundsPanicIgnoresInputSpecificValues(t *testing.T) {
	makeReport := func(message string) string {
		return "panic: " + message + `

goroutine 20 [running]:
example.com/project.lookup()
	/work/project/index.go:30 +0x10`
	}

	first := Parse("", makeReport("runtime error: index out of range [3] with length 2"))
	second := Parse("", makeReport("runtime error: index out of range [400] with length 17"))
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("unexpected event counts: first=%d second=%d", len(first), len(second))
	}
	if first[0].Signature != second[0].Signature {
		t.Fatal("input-specific index and length changed the panic signature")
	}

	userFirst := Parse("panic: invalid shard 3", "")
	userSecond := Parse("panic: invalid shard 4", "")
	if userFirst[0].Signature == userSecond[0].Signature {
		t.Fatal("normalization erased semantic numbers from a user panic")
	}
}

func TestFatalSkipsInternalRuntimeFrames(t *testing.T) {
	withRuntimeFrame := `fatal error: concurrent map writes

goroutine 8 [running]:
internal/runtime/maps.fatal({0x1, 0x2})
	/usr/local/go/src/internal/runtime/maps/fatal.go:1024 +0x5d
example.com/project.updateMap()
	/work/project/cache.go:71 +0x44`
	withoutRuntimeFrame := `fatal error: concurrent map writes

goroutine 99 [running]:
example.com/project.updateMap()
	/another/root/cache.go:71 +0xff`

	first := Parse("", withRuntimeFrame)
	second := Parse("", withoutRuntimeFrame)
	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("unexpected event counts: first=%d second=%d", len(first), len(second))
	}
	if first[0].Signature != second[0].Signature {
		t.Fatal("internal/runtime frame was selected instead of the application frame")
	}
}

func TestTimeoutAndDeadlockAreHangCandidates(t *testing.T) {
	tests := []string{
		"panic: test timed out after 30s",
		"fatal error: all goroutines are asleep - deadlock!",
	}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			events := Parse(input, "")
			if len(events) != 1 {
				t.Fatalf("Parse() returned %d events, want 1", len(events))
			}
			if events[0].Kind != KindHangCandidate {
				t.Fatalf("Kind = %q, want %q", events[0].Kind, KindHangCandidate)
			}
			if events[0].Triggered() {
				t.Fatal("hang candidate must not trigger the bug oracle")
			}
		})
	}
}

func TestPlainTimeoutSignalsAreNotBugs(t *testing.T) {
	input := "context deadline exceeded\n{TIMEOUT} {1, 2}\n{TIMEOUT_OP} {3, 4}\n"
	if events := Parse(input, "process timed out"); len(events) != 0 {
		t.Fatalf("Parse() returned timeout events as bugs: %#v", events)
	}
}

// 疑似测试顺序依赖：单独运行 _1 测试时包级共享变量未初始化，
// 栈帧中方法接收者为 0x0（如 (*BeeMap).Count(0x0)）。
const nilReceiverPanicReport = `panic: runtime error: invalid memory address or nil pointer dereference [recovered, repanicked]
[signal 0xc0000005 code=0x0 addr=0x0 pc=0x1403e7825]

goroutine 18 [running]:
testing.tRunner.func1.2({0x14057eee0, 0x14052d140})
    D:/Program Files/GO/go1.25.1/src/testing/testing.go:1872 +0x3fc
panic({0x14057eee0?, 0x14052d140?})
    D:/Program Files/GO/go1.25.1/src/runtime/panic.go:783 +0x132
github.com/beego/beego/v2/core/utils.(*BeeMap).Count(0x0)
    D:/gopath/src/real-projects/BEEGO/beegoF/core/utils/safemap.go:105 +0x65
github.com/beego/beego/v2/core/utils.TestCount_1(0xc000086380)
    D:/gopath/src/real-projects/BEEGO/beegoF/core/utils/safemap_test.go:230 +0xb3
`

func TestNilReceiverPanicClassifiedAsTestOrderSuspect(t *testing.T) {
	events := Parse("", nilReceiverPanicReport)
	if len(events) != 1 {
		t.Fatalf("Parse() returned %d events, want 1", len(events))
	}
	if events[0].Kind != KindTestOrderPanic {
		t.Fatalf("Kind = %q, want %q", events[0].Kind, KindTestOrderPanic)
	}
	if events[0].Triggered() {
		t.Fatal("test-order suspect panic must not trigger the bug oracle")
	}
}

func TestPlainNilPointerPanicStaysPanic(t *testing.T) {
	// 同样是 nil 解引用，但栈帧中无 0x0 接收者特征 → 保持普通 panic
	input := `panic: runtime error: invalid memory address or nil pointer dereference

goroutine 1 [running]:
example.com/pkg.Do(0xc0000abc)
    /work/a.go:10 +0x5
`
	events := Parse("", input)
	if len(events) != 1 {
		t.Fatalf("Parse() returned %d events, want 1", len(events))
	}
	if events[0].Kind != KindPanic {
		t.Fatalf("Kind = %q, want %q", events[0].Kind, KindPanic)
	}
	if !events[0].Triggered() {
		t.Fatal("plain nil pointer panic must still trigger the bug oracle")
	}
}
