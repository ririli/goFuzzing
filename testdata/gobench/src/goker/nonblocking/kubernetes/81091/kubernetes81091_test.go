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
	defer callstack.Trace(77309411329)()
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
	defer callstack.Trace(77309411330)()
	f := &framework{}
	f.filterPlugins = append(f.filterPlugins, &FakeFilterPlugin{})
	return f
}

func (f *framework) RunFilterPlugins() {
	defer callstack.Trace(77309411331)()
	for _, pl := range f.filterPlugins {
		pl.Filter()
	}
}

type genericScheduler struct {
	framework Framework
}

func NewGenericScheduler(framework Framework) *genericScheduler {
	defer callstack.Trace(77309411332)()
	return &genericScheduler{
		framework: framework,
	}
}

func (g *genericScheduler) findNodesThatFit() {
	defer callstack.Trace(77309411333)()
	checkNode := func(i int) {
		defer callstack.Trace(77309411334)()
		g.framework.RunFilterPlugins()
	}
	ParallelizeUntil(2, 2, checkNode)
}

func (g *genericScheduler) Schedule() {
	defer callstack.Trace(77309411335)()
	g.findNodesThatFit()
}

type DoWorkPieceFunc func(piece int)

func ParallelizeUntil(workers, pieces int, doWorkPiece DoWorkPieceFunc) {
	defer callstack.Trace(77309411336)()
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
			defer callstack.Trace(77309411337)()
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
	defer callstack.Trace(77309411338)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(77309411339)()
		defer wg.Done()
		filterFramework := NewFramework()
		scheduler := NewGenericScheduler(filterFramework)
		scheduler.Schedule()
	}()
	wg.Wait()
}
func TestKubernetes81091_1(t *testing.T) {
	defer callstack.Trace(77309411338)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(77309411339)()
		defer wg.Done()
		filterFramework := NewFramework()
		scheduler := NewGenericScheduler(filterFramework)
		scheduler.Schedule()
	}()
	wg.Wait()
}
