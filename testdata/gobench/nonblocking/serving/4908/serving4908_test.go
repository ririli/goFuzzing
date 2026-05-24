package serving4908

import (
	"sync"
	"testing"
	callstack "toolkit/pkg/callstack"
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
	defer callstack.Trace(459561500673)()
	for i := range ce.cores {
		ce.cores[i].Write()
	}
}

type testingWriter struct {
	t TestingT
}

func newTestingWriter(t TestingT) testingWriter {
	defer callstack.Trace(459561500674)()
	return testingWriter{t: t}
}

func (w testingWriter) Write() {
	defer callstack.Trace(459561500675)()
	w.t.Logf("%s", "1")
}

type Logger struct {
	core Core
}

func (log *Logger) clone() *Logger {
	defer callstack.Trace(459561500676)()
	copy := *log
	return &copy
}

func (log *Logger) Check() *CheckedEntry {
	defer callstack.Trace(459561500677)()
	ent := &CheckedEntry{}
	ent.cores = append(ent.cores, log.core)
	return ent
}

func NewLogger(t TestingT) *Logger {
	defer callstack.Trace(459561500678)()
	writer := newTestingWriter(t)
	return New(NewCore(writer))
}

func New(core Core) *Logger {
	defer callstack.Trace(459561500679)()
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
	defer callstack.Trace(459561500680)()
	c.out.Write()
}

func NewCore(ws WriteSyncer) Core {
	defer callstack.Trace(459561500681)()
	return &ioCore{
		out: ws,
	}
}

func testing_TestLogger(t *testing.T) *SugaredLogger {
	defer callstack.Trace(459561500682)()
	return NewLogger(t).Sugar()
}

func (log *Logger) Sugar() *SugaredLogger {
	defer callstack.Trace(459561500683)()
	return &SugaredLogger{log.clone()}
}

type SugaredLogger struct {
	base *Logger
}

func (s *SugaredLogger) log() {
	defer callstack.Trace(459561500684)()
	ce := s.base.Check()
	ce.Write()
}

func (s *SugaredLogger) Info(args ...interface{}) {
	defer callstack.Trace(459561500685)()
	s.log()
}

type revisionWatcher struct {
	logger *SugaredLogger
}

func newRevisionWatcher(logger *SugaredLogger) *revisionWatcher {
	defer callstack.Trace(459561500686)()
	return &revisionWatcher{
		logger: logger,
	}
}

func (rw *revisionWatcher) runWithTickCh() {
	defer callstack.Trace(459561500687)()
	rw.checkDests()
}

func (rw *revisionWatcher) checkDests() {
	defer callstack.Trace(459561500688)()
	go func() {
		defer callstack.Trace(459561500689)()
		rw.logger.Info("1")
	}()
}

func TestServing4908(t *testing.T) {
	defer callstack.Trace(459561500690)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(459561500691)()
		defer wg.Done()
		t.Run("TestServing4908", func(t *testing.T) {
			defer callstack.Trace(459561500692)()
			rw := newRevisionWatcher(
				testing_TestLogger(t),
			)
			var _wg sync.WaitGroup
			_wg.Add(1)
			go func() {
				defer callstack.Trace(459561500693)()
				rw.runWithTickCh()
				_wg.Done()
			}()
			_wg.Wait()
		})
	}()
	wg.Wait()
}
func TestServing4908_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.PrintSusConPairs()
	defer callstack.Trace(459561500690)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(459561500691)()
		defer wg.Done()
		t.Run("TestServing4908", func(t *testing.T) {
			defer callstack.Trace(459561500692)()
			rw := newRevisionWatcher(
				testing_TestLogger(t),
			)
			var _wg sync.WaitGroup
			_wg.Add(1)
			go func() {
				defer callstack.Trace(459561500693)()
				rw.runWithTickCh()
				_wg.Done()
			}()
			_wg.Wait()
		})
	}()
	wg.Wait()
}
