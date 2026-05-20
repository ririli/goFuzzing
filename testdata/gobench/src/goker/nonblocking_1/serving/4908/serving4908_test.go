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
	defer callstack.Trace(223338299393)()
	for i := range ce.cores {
		ce.cores[i].Write()
	}
}

type testingWriter struct {
	t TestingT
}

func newTestingWriter(t TestingT) testingWriter {
	defer callstack.Trace(223338299394)()
	return testingWriter{t: t}
}

func (w testingWriter) Write() {
	defer callstack.Trace(223338299395)()
	w.t.Logf("%s", "1")
}

type Logger struct {
	core Core
}

func (log *Logger) clone() *Logger {
	defer callstack.Trace(223338299396)()
	copy := *log
	return &copy
}

func (log *Logger) Check() *CheckedEntry {
	defer callstack.Trace(223338299397)()
	ent := &CheckedEntry{}
	ent.cores = append(ent.cores, log.core)
	return ent
}

func NewLogger(t TestingT) *Logger {
	defer callstack.Trace(223338299398)()
	writer := newTestingWriter(t)
	return New(NewCore(writer))
}

func New(core Core) *Logger {
	defer callstack.Trace(223338299399)()
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
	defer callstack.Trace(223338299400)()
	c.out.Write()
}

func NewCore(ws WriteSyncer) Core {
	defer callstack.Trace(223338299401)()
	return &ioCore{
		out: ws,
	}
}

func testing_TestLogger(t *testing.T) *SugaredLogger {
	defer callstack.Trace(223338299402)()
	return NewLogger(t).Sugar()
}

func (log *Logger) Sugar() *SugaredLogger {
	defer callstack.Trace(223338299403)()
	return &SugaredLogger{log.clone()}
}

type SugaredLogger struct {
	base *Logger
}

func (s *SugaredLogger) log() {
	defer callstack.Trace(223338299404)()
	ce := s.base.Check()
	ce.Write()
}

func (s *SugaredLogger) Info(args ...interface{}) {
	defer callstack.Trace(223338299405)()
	s.log()
}

type revisionWatcher struct {
	logger *SugaredLogger
}

func newRevisionWatcher(logger *SugaredLogger) *revisionWatcher {
	defer callstack.Trace(223338299406)()
	return &revisionWatcher{
		logger: logger,
	}
}

func (rw *revisionWatcher) runWithTickCh() {
	defer callstack.Trace(223338299407)()
	rw.checkDests()
}

func (rw *revisionWatcher) checkDests() {
	defer callstack.Trace(223338299408)()
	go func() {
		defer callstack.Trace(223338299409)()
		rw.logger.Info("1")
	}()
}

func TestServing4908(t *testing.T) {
	defer callstack.Trace(223338299410)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(223338299411)()
		defer wg.Done()
		t.Run("TestServing4908", func(t *testing.T) {
			defer callstack.Trace(223338299412)()
			rw := newRevisionWatcher(
				testing_TestLogger(t),
			)
			var _wg sync.WaitGroup
			_wg.Add(1)
			go func() {
				defer callstack.Trace(223338299413)()
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
	defer callstack.Trace(223338299410)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(223338299411)()
		defer wg.Done()
		t.Run("TestServing4908", func(t *testing.T) {
			defer callstack.Trace(223338299412)()
			rw := newRevisionWatcher(
				testing_TestLogger(t),
			)
			var _wg sync.WaitGroup
			_wg.Add(1)
			go func() {
				defer callstack.Trace(223338299413)()
				rw.runWithTickCh()
				_wg.Done()
			}()
			_wg.Wait()
		})
	}()
	wg.Wait()
}
