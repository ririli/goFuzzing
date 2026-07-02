package serving4908

import (
	"sync"
	"testing"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/sched"
)

type TestingT interface {
	Logf(string, ...interface{})
}

type WriteSyncer interface {
	Write()
}

type CheckedEntry struct {
	ErrorOutput WriteSyncer
	cores       []Core
}

func (ce *CheckedEntry) Write() {
	for i := range ce.cores {
		ce.cores[i].Write()
	}
}

type testingWriter struct {
	t TestingT
}

func newTestingWriter(t TestingT) testingWriter {
	return testingWriter{t: t}
}

func (w testingWriter) Write() {
	w.t.Logf("%s", "1")
}

type Logger struct {
	core Core
}

func (log *Logger) clone() *Logger {
	copy := *log
	return &copy
}

func (log *Logger) Check() *CheckedEntry {
	ent := &CheckedEntry{}
	ent.cores = append(ent.cores, log.core)
	return ent
}

func NewLogger(t TestingT) *Logger {
	writer := newTestingWriter(t)
	return New(NewCore(writer))
}

func New(core Core) *Logger {
	return &Logger{
		core: core,
	}
}

type Core interface {
	Write()
}

type ioCore struct {
	out WriteSyncer
}

func (c *ioCore) Write() {
	c.out.Write()
}

func NewCore(ws WriteSyncer) Core {
	return &ioCore{
		out: ws,
	}
}

func testing_TestLogger(t *testing.T) *SugaredLogger {
	return NewLogger(t).Sugar()
}

func (log *Logger) Sugar() *SugaredLogger {
	return &SugaredLogger{log.clone()}
}

type SugaredLogger struct {
	base *Logger
}

func (s *SugaredLogger) log() {
	ce := s.base.Check()
	ce.Write()
}

func (s *SugaredLogger) Info(args ...interface{}) {
	s.log()
}

type revisionWatcher struct {
	logger *SugaredLogger
}

func newRevisionWatcher(logger *SugaredLogger) *revisionWatcher {
	return &revisionWatcher{
		logger: logger,
	}
}

func (rw *revisionWatcher) runWithTickCh() {
	rw.checkDests()
}

func (rw *revisionWatcher) checkDests() {
	go func(_parentGid uint64) {
		goroutine.Enter(459561500673, _parentGid)
		defer goroutine.Exit(459561500673)
		func() {
			rw.logger.Info("1")
		}()
	}(goroutine.CurrentGid())
}

func TestServing4908(t *testing.T) {
	var wg sync.WaitGroup
	sched.InstWgBF(459561500676)
	wg.Add(1)
	sched.InstWgAF(459561500676, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(459561500674, _parentGid)
		defer goroutine.Exit(459561500674)
		func() {
			defer func() {
				sched.InstWgBF(459561500677)
				wg.Done()
				sched.InstWgAF(459561500677, &wg, "done")
			}()
			t.Run("TestServing4908", func(t *testing.T) {
				rw := newRevisionWatcher(
					testing_TestLogger(t),
				)
				var _wg sync.WaitGroup
				sched.InstWgBF(459561500678)
				_wg.Add(1)
				sched.InstWgAF(459561500678, &_wg, "add")
				go func(_parentGid uint64) {
					goroutine.Enter(459561500675, _parentGid)
					defer goroutine.Exit(459561500675)
					func() {
						rw.runWithTickCh()
						sched.InstWgBF(459561500679)
						_wg.Done()
						sched.InstWgAF(459561500679, &_wg, "done")
					}()
				}(goroutine.CurrentGid())
				_wg.Wait()
			})
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
func TestServing4908_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	sched.InstWgBF(459561500676)
	wg.Add(1)
	sched.InstWgAF(459561500676, &wg, "add")
	go func(_parentGid uint64) {
		goroutine.Enter(459561500674, _parentGid)
		defer goroutine.Exit(459561500674)
		func() {
			defer func() {
				sched.InstWgBF(459561500677)
				wg.Done()
				sched.InstWgAF(459561500677, &wg, "done")
			}()
			t.Run("TestServing4908", func(t *testing.T) {
				rw := newRevisionWatcher(
					testing_TestLogger(t),
				)
				var _wg sync.WaitGroup
				sched.InstWgBF(459561500678)
				_wg.Add(1)
				sched.InstWgAF(459561500678, &_wg, "add")
				go func(_parentGid uint64) {
					goroutine.Enter(459561500675, _parentGid)
					defer goroutine.Exit(459561500675)
					func() {
						rw.runWithTickCh()
						sched.InstWgBF(459561500679)
						_wg.Done()
						sched.InstWgAF(459561500679, &_wg, "done")
					}()
				}(goroutine.CurrentGid())
				_wg.Wait()
			})
		}()
	}(goroutine.CurrentGid())
	wg.Wait()
}
