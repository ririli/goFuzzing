package fuzzer

import (
	"fmt"
	_ "net/http/pprof"
	"sync/atomic"
	"time"
	"toolkit/pkg/bug"
	"toolkit/pkg/feedback"
)

var (
	debug  = false
	info   = false
	normal = true
)

type Monitor struct {
	etimes int32
	max    int32
	doinit uint32
	bugs   *bug.Set
}

type RunContext struct {
	In      Input
	Out     Output
	timeout bool
}

var workerID uint32

type bugFinding struct {
	event  bug.Event
	record bug.Record
	isNew  bool
}

type runAnalysis struct {
	gortSignals      []*feedback.CoverageSignal
	opSignals        []*feedback.CoverageSignal
	findings         []bugFinding
	newOracleFinding bool
	triggered        bool
}

func analyzeRun(ctx RunContext, executionID uint64, bugs *bug.Set) runAnalysis {
	gortSignals, opSignals := feedback.ParseSignals(ctx.Out.O)
	result := runAnalysis{
		gortSignals: gortSignals,
		opSignals:   opSignals,
	}

	evidence := bug.Evidence{
		ExecutionID: executionID,
		Mode:        "preexec",
		GortCovered: coveredPairs(gortSignals),
		OpCovered:   coveredPairs(opSignals),
		Duration:    ctx.Out.Time,
	}
	hasInput := false
	if ctx.In.pairInput != nil && !ctx.In.pairInput.IsEmpty() {
		evidence.Mode = "validate"
		evidence.GortInput = ctx.In.pairInput.ToString()
		hasInput = true
	} else if ctx.In.gortPair != nil && len(ctx.In.gortPair.TryPair) > 0 {
		evidence.Mode = "validate"
		evidence.GortInput = ctx.In.gortPair.ToString()
		hasInput = true
	} else if ctx.In.funcPair != nil && len(ctx.In.funcPair.TryPair) > 0 {
		evidence.Mode = "validate"
		evidence.GortInput = ctx.In.funcPair.ToString()
		hasInput = true
	}
	if hasInput {
		evidence.Associated = matchesCoveredInput(ctx.In, gortSignals, opSignals)
	}
	if ctx.In.tryOpPair != nil {
		evidence.OpInput = ctx.In.tryOpPair.ToString()
	}
	if ctx.Out.Err != nil {
		evidence.ExitError = ctx.Out.Err.Error()
	}

	for _, event := range bug.Parse(ctx.Out.O, ctx.Out.Trace) {
		record, isNew := bugs.Add(event, evidence)
		result.findings = append(result.findings, bugFinding{
			event:  event,
			record: record,
			isNew:  isNew,
		})
		// Novel hang candidates are retained as oracle progress even though they
		// do not make the run fail until replay confirmation is implemented.
		result.newOracleFinding = result.newOracleFinding || isNew
		result.triggered = result.triggered || event.Triggered()
	}
	return result
}

func coveredPairs(signals []*feedback.CoverageSignal) []bug.Pair {
	pairs := make([]bug.Pair, 0, len(signals))
	for _, signal := range signals {
		if signal == nil || !signal.Success {
			continue
		}
		pairs = append(pairs, bug.Pair{PreID: signal.PreID, NextID: signal.NextID})
	}
	return pairs
}

func matchesCoveredInput(in Input, gortSignals, opSignals []*feedback.CoverageSignal) bool {
	// Check unified pairInput first, then fall back to legacy fields
	matchGort := func() bool {
		if in.pairInput != nil && !in.pairInput.IsEmpty() {
			for _, signal := range gortSignals {
				if signal == nil || !signal.Success {
					continue
				}
				for _, pair := range in.pairInput.Pairs {
					if pair != nil && gortSignalKey(pair.ID1(), pair.ID2()) == gortSignalKey(signal.PreID, signal.NextID) {
						return true
					}
				}
			}
		} else if in.gortPair != nil && len(in.gortPair.TryPair) > 0 {
			for _, signal := range gortSignals {
				if signal == nil || !signal.Success {
					continue
				}
				for _, pair := range in.gortPair.TryPair {
					if pair != nil && gortSignalKey(pair.Gid1, pair.Gid2) == gortSignalKey(signal.PreID, signal.NextID) {
						return true
					}
				}
			}
		} else if in.funcPair != nil && len(in.funcPair.TryPair) > 0 {
			for _, signal := range gortSignals {
				if signal == nil || !signal.Success {
					continue
				}
				for _, pair := range in.funcPair.TryPair {
					if pair != nil && gortSignalKey(pair.FuncID1, pair.FuncID2) == gortSignalKey(signal.PreID, signal.NextID) {
						return true
					}
				}
			}
		}
		return false
	}
	if gortOk := matchGort(); gortOk {
		return true
	}
	if in.tryOpPair == nil {
		return false
	}
	for _, signal := range opSignals {
		if signal == nil || !signal.Success {
			continue
		}
		for _, pair := range in.tryOpPair.TryPair {
			if pair == nil || pair.Op1 == nil || pair.Op2 == nil {
				continue
			}
			if opSignalKey(pair.Op1.OpId, pair.Op2.OpId) == opSignalKey(signal.PreID, signal.NextID) {
				return true
			}
		}
	}
	return false
}

func monitorResult(bugs *bug.Set) (bool, []string) {
	if bugs != nil && bugs.HasTriggered() {
		return true, []string{"FAIL", bugs.Summary()}
	}
	return false, []string{"PASS", ""}
}

func shouldStopAfterRun(singleCrash bool, analysis runAnalysis) bool {
	return singleCrash && analysis.triggered
}

func sendMonitorLog(logCh chan<- string, message string) {
	if logCh == nil {
		return
	}
	logCh <- message
}

func logBugFindings(logCh chan<- string, wid uint32, executionID uint64, findings []bugFinding) {
	for _, finding := range findings {
		if normal {
			sendMonitorLog(logCh, fmt.Sprintf(
				"%s\t[WORKER %v] ORACLE kind=%s signature=%s new=%t count=%d associated=%t execution=%d message=%q",
				time.Now().String(), wid, finding.event.Kind, finding.event.Signature,
				finding.isNew, finding.record.Count, finding.record.Last.Associated,
				executionID, finding.event.Message))
		}
		if (finding.isNew || debug) && finding.event.Report != "" {
			sendMonitorLog(logCh, fmt.Sprintf("%s\t[ORACLE REPORT] kind=%s signature=%s\n%s",
				time.Now().String(), finding.event.Kind, finding.event.Signature, finding.event.Report))
		}
	}
}

func (m *Monitor) Start(cfg *Config, ticket chan struct{}) (bool, []string) {
	startTime := time.Now()
	defer func() {
		fmt.Printf("[FUZZER] %s elapsed: %.3fs, etimes=%d\n", cfg.Fn, time.Since(startTime).Seconds(), atomic.LoadInt32(&m.etimes))
	}()
	if m.max == int32(0) {
		m.max = int32(cfg.MaxExecution)
	}
	m.doinit = uint32(1)
	m.bugs = bug.NewSet()
	switch cfg.LogLevel {
	case "debug":
		debug = true
		info = true
	case "info":
		info = true
	default:
	}

	// 根据颗粒度创建 PairCorpus 实现
	adapter := GetAdapter(cfg.Granularity)
	var pairCorpus PairCorpus
	if cfg.Granularity == ModeFunction {
		pairCorpus = NewCorpusFunc(&cfg.GortPhase)
	} else {
		pairCorpus = NewCorpusGort(&cfg.GortPhase)
	}

	corpusOp := NewCorpusOp(&cfg.GortPhase)
	wid := atomic.AddUint32(&workerID, 1)
	ch := make(chan RunContext)
	cancel := make(chan struct{})
	quit := cfg.MaxQuit
	dowork := func() {
		timeoutTimer := time.NewTimer(1 * time.Minute)
		defer timeoutTimer.Stop()
		for {
			timeoutTimer.Reset(1 * time.Minute)
			select {
			case <-cancel:
				fmt.Println("cancel and return1")
				return
			default:
			}

			pairInput := pairCorpus.GetInput()
			opPair := corpusOp.Get()
			e := Executor{}
			in := Input{
				pairInput:      pairInput,
				tryOpPair:      opPair,
				cmd:            cfg.Bin,
				args:           []string{"-test.v", "-test.run", cfg.Fn},
				timeout:        cfg.TimeOut,
				recovertimeout: cfg.RecoverTimeOut,
			}

			var istimeout bool
			done := make(chan int)
			var o *Output
			go func() {
				t := e.Run(in)
				o = &t
				close(done)
			}()
			select {
			case <-done:
			case <-timeoutTimer.C:
				istimeout = true
			case <-cancel:
				fmt.Println("cancel and return done")
				return
			}
			if o == nil {
				continue
			}
			if debug {
				sendMonitorLog(cfg.LogCh, fmt.Sprintf("%s\t[EXECUTOR] Finish, USE %s", time.Now().String(), o.Time.String()))
			}
			select {
			case <-cancel:
				fmt.Println("cancel and return before send")
				return
			case ch <- RunContext{In: in, Out: *o, timeout: istimeout}:
			}
		}
	}
	cfg.MaxWorker = 4
	for i := 0; i < cfg.MaxWorker; i++ {
		go dowork()
	}
	fmt.Println("m.max=", m.max)
	for {
		fmt.Println("m.etimes=", m.etimes)
		if m.etimes >= m.max {
			close(cancel)
			return monitorResult(m.bugs)
		}
		ctx := <-ch
		executionID := uint64(atomic.AddInt32(&m.etimes, 1))
		var inputc string

		inputc = "empty chain"

		if debug {
			sendMonitorLog(cfg.LogCh, fmt.Sprintf("%s\t[WORKER %v] Input: %s", time.Now().String(), wid, inputc))
		}

		analysis := analyzeRun(ctx, executionID, m.bugs)
		logBugFindings(cfg.LogCh, wid, executionID, analysis.findings)

		// stderr → 种子信息（通过 GranularityAdapter 解析，按颗粒度分发）
		pairs, opInfos, err := adapter.ParsePairs(ctx.Out.Trace)
		if err == nil {
			if len(pairs) > 0 {
				pairCorpus.AddConcurrencyPairs(pairs)
			}
			if len(opInfos) > 0 {
				corpusOp.Add(opInfos)
			}
		}
		edges, err := adapter.ParseEdges(ctx.Out.Trace)
		if len(edges) > 0 {
			pairCorpus.AddConcurrencyEdges(edges)
		}
		if err != nil && debug {
			sendMonitorLog(cfg.LogCh, fmt.Sprintf("%s\t[WORKER %v] Failed to parse some topology edges: %v", time.Now().String(), wid, err))
		}

		// 预执行阶段判断（通过接口统一调用）
		if atomic.LoadUint32(&cfg.GortPhase) == 0 {
			pairCorpus.TryEndPreExec(cfg.MaxPreExecRound)
			if gortCorpus, ok := pairCorpus.(*CorpusGort); ok {
				corpusOp.TryEndPreExec(gortCorpus)
			} else {
				corpusOp.TryEndPreExecForFunc(nil)
			}
		}

		// fuzzing 阶段
		gortSignals, opSingnals := analysis.gortSignals, analysis.opSignals
		fmt.Println("====stdout_start=====")
		fmt.Println(ctx.Out.O)
		fmt.Println("=======stdout_end=========")
		fmt.Println("====反馈信号_start=====")
		for _, signal := range gortSignals {
			fmt.Println(signal.PreID, signal.NextID, signal.Success, signal.Kind)
		}
		for _, singnal := range opSingnals {
			fmt.Println(singnal.PreID, singnal.NextID, singnal.Success, singnal.Kind)
		}
		fmt.Println("=======反馈信号_end=======")
		madeProgress := analysis.newOracleFinding
		if cfg.UseMutate {
			if len(gortSignals) > 0 {
				newlyCovered := pairCorpus.ApplyConcurrencySignals(gortSignals)
				if len(newlyCovered) > 0 {
					madeProgress = true
					pairCorpus.OnCoveredByOp(corpusOp, newlyCovered)
				}
			}
			if len(opSingnals) > 0 {
				corpusOp.ApplySignals(opSingnals)
			}
		}

		if madeProgress {
			quit = cfg.MaxQuit
		} else {
			quit--
		}
		if shouldStopAfterRun(cfg.SingleCrash, analysis) {
			close(cancel)
			return monitorResult(m.bugs)
		}
		fmt.Println("quit=", quit)
		if quit <= 0 {
			if info {
				sendMonitorLog(cfg.LogCh, fmt.Sprintf("%s\t[WORKER %v] Fuzzing seems useless, QUIT", time.Now().String(), wid))
			}
			close(cancel)
			fmt.Println("exit loop")
			return monitorResult(m.bugs)
		}
	}

}
