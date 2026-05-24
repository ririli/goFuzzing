package kubernetes81091

import (
	"sync"
	"testing"
	callstack "toolkit/pkg/callstack"
)

type FakeFilterPlugin struct {
	numFilterCalled int
}

func (fp *FakeFilterPlugin) Filter() {
	defer callstack.Trace(1035087118337)()
	fp.numFilterCalled++
}

type FilterPlugin interface {
	Filter()
}

type Framework interface {
	RunFilterPlugins()
}

type framework struct {
	filterPlugins []FilterPlugin
}

func NewFramework() Framework {
	defer callstack.Trace(1035087118338)()
	f := &framework{}
	f.filterPlugins = append(f.filterPlugins, &FakeFilterPlugin{})
	return f
}

func (f *framework) RunFilterPlugins() {
	defer callstack.Trace(1035087118339)()
	for _, pl := range f.filterPlugins {
		pl.Filter()
	}
}

type genericScheduler struct {
	framework Framework
}

func NewGenericScheduler(framework Framework) *genericScheduler {
	defer callstack.Trace(1035087118340)()
	return &genericScheduler{
		framework: framework,
	}
}

func (g *genericScheduler) findNodesThatFit() {
	defer callstack.Trace(1035087118341)()
	checkNode := func(i int) {
		defer callstack.Trace(1035087118342)()
		g.framework.RunFilterPlugins()
	}
	ParallelizeUntil(2, 2, checkNode)
}

func (g *genericScheduler) Schedule() {
	defer callstack.Trace(1035087118343)()
	g.findNodesThatFit()
}

type DoWorkPieceFunc func(piece int)

func ParallelizeUntil(workers, pieces int, doWorkPiece DoWorkPieceFunc) {
	defer callstack.Trace(1035087118344)()
	var stop <-chan struct{}

	toProcess := make(chan int, pieces)
	for i := 0; i < pieces; i++ {
		toProcess <- i
	}
	close(toProcess)

	if pieces < workers {
		workers = pieces
	}

	wg := sync.WaitGroup{}
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer callstack.Trace(1035087118345)()
			defer wg.Done()
			for piece := range toProcess {
				select {
				case <-stop:
					return
				default:
					doWorkPiece(piece)
				}
			}
		}()
	}
	wg.Wait()
}

func TestKubernetes81091(t *testing.T) {
	defer callstack.Trace(1035087118346)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(1035087118347)()
		defer wg.Done()
		filterFramework := NewFramework()
		scheduler := NewGenericScheduler(filterFramework)
		scheduler.Schedule()
	}()
	wg.Wait()
}
func TestKubernetes81091_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(1035087118346)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(1035087118347)()
		defer wg.Done()
		filterFramework := NewFramework()
		scheduler := NewGenericScheduler(filterFramework)
		scheduler.Schedule()
	}()
	wg.Wait()
}
