package serving6171

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

func (s *SugaredLogger) Errorw(args ...interface{}) {
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

func (rw *revisionWatcher) run() {
	rw.checkDests()
}

func (rw *revisionWatcher) checkDests() {
	go func() {
		goroutine.Enter(468151435265)
		defer goroutine.Exit(468151435265)
		func() {
			rw.logger.Errorw("1")
		}()
	}()
}

type revisionBackendsManager struct {
	logger *SugaredLogger
}

func (rbm *revisionBackendsManager) getOrCreateRevisionWatcher() {
	rw := newRevisionWatcher(rbm.logger)
	go func() {
		goroutine.Enter(468151435266)
		defer goroutine.Exit(468151435266)
		rw.run()
	}()
}

func TestServing6171(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		goroutine.Enter(468151435267)
		defer goroutine.Exit(468151435267)
		func() {
			defer wg.Done()
			t.Run("Serving6171", func(t *testing.T) {
				rbm := &revisionBackendsManager{logger: testing_TestLogger(t)}
				rbm.getOrCreateRevisionWatcher()
			})
		}()
	}()
	wg.Wait()
}
func TestServing6171_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		goroutine.Enter(468151435267)
		defer goroutine.Exit(468151435267)
		func() {
			defer wg.Done()
			t.Run("Serving6171", func(t *testing.T) {
				rbm := &revisionBackendsManager{logger: testing_TestLogger(t)}
				rbm.getOrCreateRevisionWatcher()
			})
		}()
	}()
	wg.Wait()
}
