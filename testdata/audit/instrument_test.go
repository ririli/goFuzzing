package audit

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"toolkit/pkg/inst"
	"toolkit/pkg/inst/passes"
)

func TestInstrumentedGoAndTestSemantics(t *testing.T) {
	// Generated source stays in testdata; the original benchmarks are untouched.
	dir, err := os.MkdirTemp(".", ".work.instrument-")
	if err != nil {
		t.Fatal(err)
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "sample_test.go")
	source := `package sample
import ("testing"; "sync")
type counter struct { n int }
func (c *counter) increment(done chan struct{}) { c.n++; close(done) }
func TestSemantics(check *testing.T) {
 var c counter
 done := make(chan struct{})
 go c.increment(done)
 <-done
 if c.n != 1 { check.Fatal("receiver was copied") }
 stop := make(chan struct{})
 result := make(chan int, 1)
 f := func() { <-stop; result <- 1 }
 go f()
 f = func() { result <- 2 }
 close(stop)
 if <-result != 1 { check.Fatal("function value was evaluated in child") }
 numbers := make(chan uint64, 1)
 send := func(v uint64) { numbers <- v }
 go send(1)
 if <-numbers != 1 { check.Fatal("constant changed") }
 selected := make(chan int, 1)
 evaluations := 0
 channel := func() chan int { evaluations++; return selected }
 select { case channel() <- 1: default: }
 if evaluations != 1 { check.Fatal("select channel evaluated twice") }
 var wg sync.WaitGroup
 receivers := 0
 group := func() *sync.WaitGroup { receivers++; return &wg }
 group().Add(1)
 if receivers != 1 { check.Fatal("WaitGroup receiver evaluated twice") }
 wg.Done()
 func() {
   delta := 1
   defer wg.Add(delta)
   delta = -1
 }()
 wg.Done()
 wg.Wait()
}
`
	if err := os.WriteFile(file, []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	ctx, err := inst.NewInstContext(file)
	if err != nil {
		t.Fatal(err)
	}
	inst.RunPass(&passes.GoroutinePass{}, ctx)
	inst.RunPass(&passes.ChRecPass{}, ctx)
	inst.RunPass(&passes.SelectPass{}, ctx)
	inst.RunPass(&passes.WgPass{}, ctx)
	inst.RunPass(&passes.TestPass{Granularity: "goroutine"}, ctx)
	if err := inst.DumpAstFile(ctx.FS, ctx.AstFile, file); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "test", "-race", "-count=1", "-run=^TestSemantics_1$", ".")
	command.Dir = dir
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("instrumented program: %v\n%s", err, out)
	}
}
