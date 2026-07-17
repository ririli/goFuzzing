package fuzzer

import (
	"fmt"
	_ "net/http/pprof"
	"strings"
	"sync/atomic"
	"time"
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
}

type RunContext struct {
	In      Input
	Out     Output
	timeout bool
}

var workerID uint32

func (m *Monitor) Start(cfg *Config, ticket chan struct{}) (bool, []string) {
	startTime := time.Now()
	defer func() {
		fmt.Printf("[FUZZER] %s elapsed: %.3fs, etimes=%d\n", cfg.Fn, time.Since(startTime).Seconds(), atomic.LoadInt32(&m.etimes))
	}()
	if m.max == int32(0) {
		m.max = int32(cfg.MaxExecution)
	}
	m.doinit = uint32(1)
	switch cfg.LogLevel {
	case "debug":
		debug = true
		info = true
	case "info":
		info = true
	default:
	}

	var corpusGort *CorpusGort
	corpusGort = NewCorpusGort(&cfg.GortPhase)
	var corpusOp *CorpusOp
	corpusOp = NewCorpusOp(&cfg.GortPhase)
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

			gortPair := corpusGort.Get()
			opPair := corpusOp.Get()
			e := Executor{}
			in := Input{
				gortPair:       gortPair,
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
				cfg.LogCh <- fmt.Sprintf("%s\t[EXECUTOR] Finish, USE %s", time.Now().String(), o.Time.String())
			}
			select {
			case <-cancel:
				fmt.Println("cancel and return before send")
				return
			default:
				ch <- RunContext{In: in, Out: *o, timeout: istimeout}
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
			return false, []string{}
		}
		ctx := <-ch
		atomic.AddInt32(&m.etimes, 1)
		var inputc string

		inputc = "empty chain"

		if debug {
			cfg.LogCh <- fmt.Sprintf("%s\t[WORKER %v] Input: %s", time.Now().String(), wid, inputc)
		}

		// panic收集
		if strings.Contains(ctx.Out.O, "panic:") || strings.Contains(ctx.Out.Trace, "panic:") {
			panicOutput := ctx.Out.O
			if !strings.Contains(panicOutput, "panic:") {
				panicOutput = ctx.Out.Trace
			}
			if idx := strings.Index(panicOutput, "panic:"); idx != -1 {
				panicMsg := panicOutput[idx:]
				if normal {
					cfg.LogCh <- fmt.Sprintf("%s\t[WORKER %v] PANIC [%v]\n%s", time.Now().String(), wid, atomic.LoadInt32(&m.etimes), panicMsg)
				}
				if debug {
					cfg.LogCh <- fmt.Sprintf("%s\t[PANIC DEBUG] Full Trace:\n%s", time.Now().String(), panicMsg)
				}
			}
		}
		if strings.Contains(ctx.Out.Trace, "WARNING: DATA RACE") {
			raceReport := ctx.Out.Trace
			if normal {
				cfg.LogCh <- fmt.Sprintf("%s\t[WORKER %v] RACE DETECTED [%v]\n%s", time.Now().String(), wid, atomic.LoadInt32(&m.etimes), raceReport)
			}
			if debug {
				cfg.LogCh <- fmt.Sprintf("%s\t[RACE DEBUG] Full Trace:\n%s", time.Now().String(), raceReport)
			}
		}
		// stderr → 种子信息
		pair_st, opInfos, err := feedback.ParseGortPairs(ctx.Out.Trace)
		if err == nil {
			if len(pair_st) > 0 {
				corpusGort.AddPair(pair_st)
			}
			if len(opInfos) > 0 {
				corpusOp.Add(opInfos)
			}
		}
		edges, err := feedback.ParseGortEdges(ctx.Out.Trace)
		if len(edges) > 0 {
			corpusGort.AddEdges(edges)
		}
		if err != nil && debug {
			cfg.LogCh <- fmt.Sprintf("%s\t[WORKER %v] Failed to parse some goroutine topology edges: %v", time.Now().String(), wid, err)
		}

		// 预执行阶段判断
		if atomic.LoadUint32(&cfg.GortPhase) == 0 {
			corpusGort.TryEndPreExec(cfg.MaxPreExecRound)
			corpusOp.TryEndPreExec(corpusGort)
		}

		// fuzzing 阶段
		gortSignals, opSingnals := feedback.ParseSignals(ctx.Out.O)
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
		madeProgress := false
		if cfg.UseMutate {
			if len(gortSignals) > 0 {
				newlyCovered := corpusGort.ApplySignals(gortSignals)
				if len(newlyCovered) > 0 {
					madeProgress = true
					corpusOp.OnGortCovered(newlyCovered)
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
		fmt.Println("quit=", quit)
		if quit <= 0 {
			if info {
				cfg.LogCh <- fmt.Sprintf("%s\t[WORKER %v] Fuzzing seems useless, QUIT", time.Now().String(), wid)
			}
			close(cancel)
			fmt.Println("exit loop")
			return false, []string{}
		}
	}

}
