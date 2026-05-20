package serving6171

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
	defer callstack.Trace(738734374913)()
	for i := range ce.cores {
		ce.cores[i].Write()
	}
}

type testingWriter struct {
	t TestingT
}

func newTestingWriter(t TestingT) testingWriter {
	defer callstack.Trace(738734374914)()
	return testingWriter{t: t}
}

func (w testingWriter) Write() {
	defer callstack.Trace(738734374915)()
	w.t.Logf("%s", "1")
}

type Logger struct {
	core Core
}

func (log *Logger) clone() *Logger {
	defer callstack.Trace(738734374916)()
	copy := *log
	return &copy
}

func (log *Logger) Check() *CheckedEntry {
	defer callstack.Trace(738734374917)()
	ent := &CheckedEntry{}
	ent.cores = append(ent.cores, log.core)
	return ent
}

func NewLogger(t TestingT) *Logger {
	defer callstack.Trace(738734374918)()
	writer := newTestingWriter(t)
	return New(NewCore(writer))
}

func New(core Core) *Logger {
	defer callstack.Trace(738734374919)()
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
	defer callstack.Trace(738734374920)()
	c.out.Write()
}

func NewCore(ws WriteSyncer) Core {
	defer callstack.Trace(738734374921)()
	return &ioCore{
		out: ws,
	}
}

func testing_TestLogger(t *testing.T) *SugaredLogger {
	defer callstack.Trace(738734374922)()
	return NewLogger(t).Sugar()
}

func (log *Logger) Sugar() *SugaredLogger {
	defer callstack.Trace(738734374923)()
	return &SugaredLogger{log.clone()}
}

type SugaredLogger struct {
	base *Logger
}

func (s *SugaredLogger) log() {
	defer callstack.Trace(738734374924)()
	ce := s.base.Check()
	ce.Write()
}

func (s *SugaredLogger) Errorw(args ...interface{}) {
	defer callstack.Trace(738734374925)()
	s.log()
}

type revisionWatcher struct {
	logger *SugaredLogger
}

func newRevisionWatcher(logger *SugaredLogger) *revisionWatcher {
	defer callstack.Trace(738734374926)()
	return &revisionWatcher{
		logger: logger,
	}
}

func (rw *revisionWatcher) run() {
	defer callstack.Trace(738734374927)()
	rw.checkDests()
}

func (rw *revisionWatcher) checkDests() {
	defer callstack.Trace(738734374928)()
	go func() {
		defer callstack.Trace(738734374929)()
		rw.logger.Errorw("1")
	}()
}

type revisionBackendsManager struct {
	logger *SugaredLogger
}

func (rbm *revisionBackendsManager) getOrCreateRevisionWatcher() {
	defer callstack.Trace(738734374930)()
	rw := newRevisionWatcher(rbm.logger)
	go rw.run()
}

func TestServing6171(t *testing.T) {
	defer callstack.Trace(738734374931)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(738734374932)()
		defer wg.Done()
		t.Run("Serving6171", func(t *testing.T) {
			defer callstack.Trace(738734374933)()
			rbm := &revisionBackendsManager{logger: testing_TestLogger(t)}
			rbm.getOrCreateRevisionWatcher()
		})
	}()
	wg.Wait()
}
func TestServing6171_1(t *testing.T) {
	callstack.ParseInput()
	defer callstack.Trace(738734374931)()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer callstack.Trace(738734374932)()
		defer wg.Done()
		t.Run("Serving6171", func(t *testing.T) {
			defer callstack.Trace(738734374933)()
			rbm := &revisionBackendsManager{logger: testing_TestLogger(t)}
			rbm.getOrCreateRevisionWatcher()
		})
	}()
	wg.Wait()
}
