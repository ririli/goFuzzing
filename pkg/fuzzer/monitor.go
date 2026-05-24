package fuzzer

import (
	"fmt"
	_ "net/http/pprof"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"toolkit/pkg/bug"
	"toolkit/pkg/feedback"
	"toolkit/pkg/seed"
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

func (m *Monitor) Start(cfg *Config, visitor *Visitor, ticket chan struct{}) (bool, []string) {
	//log.Println(http.ListenAndServe(":6060", nil))
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
	var fncov *feedback.Cov
	var corpus *Corpus
	// todo 实现输入和输出
	var corpusPair *CorpusPair
	corpusPair = NewCorpusPair()
	var maxscore *int32

	if visitor.V_cov == nil {
		fncov = feedback.NewCov()
	} else {
		fncov = visitor.V_cov
	}

	if visitor.V_corpus == nil {
		corpus = NewCorpus()
	} else {
		corpus = visitor.V_corpus
	}

	if visitor.V_score == nil {
		score := int32(10)
		maxscore = &score
	} else {
		maxscore = visitor.V_score
	}

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
			var c, ht *Chain
			c, ht = corpus.Get()
			tryPair := corpusPair.Get()
			if !cfg.UseMutate || atomic.LoadUint32(&m.doinit) == uint32(1) { // if no feedback, no seed and mutation
				c = nil
				ht = nil
			}
			e := Executor{}
			in := Input{
				tryPair: tryPair,
				c:       c,
				ht:      ht,
				cmd:     cfg.Bin,
				args:    []string{"-test.v", "-test.run", cfg.Fn},
				// args:           []string{"-test.v", "-test.run", cfg.Fn, "-test.timeout", "30s"},
				timeout:        cfg.TimeOut,
				recovertimeout: cfg.RecoverTimeOut,
			}
			// atomic.AddInt32(&m.etimes, 1)

			//timeout := time.After(1 * time.Minute)
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
			case <-cancel: // 增加对 cancel 的监听
				fmt.Println("cancel and return done")
				return
			}
			//select {
			//case <-done:
			//case <-timeout:
			//	istimeout = true
			//}
			// if ok {
			//	ticket <- struct{}{}
			//}
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
			//ch <- RunContext{In: in, Out: *o, timeout: istimeout}
			//select {
			//case <-cancel:
			//	break
			//default:
			//}
		}
	}
	for i := 0; i < cfg.MaxWorker; i++ {
		go dowork()
	}
	fmt.Println("m.max=", m.max)
	for {
		fmt.Println("m.etimes=", m.etimes)
		if m.etimes > m.max {
			close(cancel)
			return false, []string{}
		}
		ctx := <-ch // 接收worker的执行结果
		atomic.AddInt32(&m.etimes, 1)
		var inputc string
		if ctx.In.c != nil {
			inputc = ctx.In.c.ToString()
		} else {
			inputc = "empty chain"
		}
		if debug {
			cfg.LogCh <- fmt.Sprintf("%s\t[WORKER %v] Input: %s", time.Now().String(), wid, inputc)
		}
		// global corpus is not thread safe now
		if ctx.Out.Err != nil {
			// ignore normal test fail
			if ctx.Out.Time < time.Duration(cfg.TimeOut)*time.Second &&
				(strings.Contains(ctx.Out.O, "panic") || strings.Contains(ctx.Out.O, "found unexpected goroutines") ||
					strings.Contains(ctx.Out.Trace, "all goroutines are asleep - deadlock!")) {
				tfs := bug.TopF(ctx.Out.O)
				exist := cfg.BugSet.Exist(tfs, cfg.Fn)
				if !exist {
					detail := []string{inputc, strconv.FormatInt(int64(atomic.LoadInt32(&m.etimes)), 10), ctx.Out.O}
					if normal {
						if strings.Contains(ctx.Out.Trace, "all goroutines are asleep - deadlock!") {
							cfg.LogCh <- fmt.Sprintf("%s\t[WORKER %v] CRASH [%v] \n %s", time.Now().String(), cfg.BugSet.Size(), inputc, "all goroutines are asleep - deadlock!")
						} else {
							cfg.LogCh <- fmt.Sprintf("%s\t[WORKER %v] CRASH [%v] \n %s", time.Now().String(), cfg.BugSet.Size(), inputc, ctx.Out.O)
						}
					}
					if debug {
						topfs := ""
						for _, f := range tfs {
							topfs += f + "\n"
						}
						cfg.LogCh <- fmt.Sprintf("%s\t[BUG] [%s] TopF : \n%s", time.Now().String(), cfg.Fn, topfs)
					}
					if cfg.SingleCrash {
						fmt.Println("cfg.SingleCrash")
						close(cancel)
						return true, detail
					}
				}
			}

		}
		// panic收集：捕获 Go runtime panic 并输出完整堆栈
		if strings.Contains(ctx.Out.O, "panic:") || strings.Contains(ctx.Out.Trace, "panic:") {
			// 优先从 stdout 取，否则从 stderr 取
			panicOutput := ctx.Out.O
			if !strings.Contains(panicOutput, "panic:") {
				panicOutput = ctx.Out.Trace
			}
			// 截取从 "panic:" 开始的所有内容（包含完整堆栈）
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
		// ✅ 新增：专门处理 -race 输出的逻辑
		if strings.Contains(ctx.Out.Trace, "WARNING: DATA RACE") {
			raceReport := ctx.Out.Trace
			if normal {
				cfg.LogCh <- fmt.Sprintf("%s\t[WORKER %v] RACE DETECTED [%v]\n%s", time.Now().String(), wid, atomic.LoadInt32(&m.etimes), raceReport)
			}
			if debug {
				cfg.LogCh <- fmt.Sprintf("%s\t[RACE DEBUG] Full Trace:\n%s", time.Now().String(), raceReport)
			}
			// 如果希望发现 Race 就停止，可以取消下面的注释
			// close(cancel)
			// return true, []string{inputc, "DATA RACE", raceReport}
		}
		op_st, all := feedback.ParseLog(ctx.Out.Trace)
		schedcov := feedback.ParseCovered(ctx.Out.O)
		schedres, coveredinput := ColorCovered(ctx.Out.O, ctx.In.c)

		pair_st, err := feedback.ParseStdPairs(ctx.Out.Trace)
		if err == nil && len(pair_st) > 0 {

			corpusPair.AddPair(pair_st)
		}
		cov := feedback.Log2Cov(op_st, all)
		score := cov.Score(cfg.UseStates)
		// if len(schedcov) != 0 && cfg.UseCoveredSched {
		//	score += (len(schedcov) / (ctx.In.c.Len())) * len(schedcov) * 10
		// }
		curmax := atomic.LoadInt32(maxscore)
		if int32(score) > curmax {
			atomic.StoreInt32(maxscore, int32(score))
		}
		energy := int(float64(score+1) / float64(curmax) * 100)
		if debug {
			cfg.LogCh <- fmt.Sprintf("%s\t[WORKER %v] score : %v\tenergy %v", time.Now().String(), wid, score, energy)
		}

		init := atomic.LoadUint32(&m.doinit) == 1
		if init && atomic.LoadInt32(&m.etimes) > int32(cfg.InitTurnCnt) {
			atomic.StoreUint32(&m.doinit, 0)
			if cfg.UseMutate {
				fmt.Printf("[MUTATE] SWITCH TO MUTATION MODE, CURRENT INITCNT %v\n", cfg.InitTurnCnt)
			}
		}
		if init {
			if cfg.UseFeedBack {
				go func() { // static analysis at a single routine
					seeds := seed.SRDOAnalysis(op_st)
					seeds = append(seeds, seed.SODRAnalysis(op_st)...)
					if debug {
						if len(seeds) != 0 {
							cfg.LogCh <- fmt.Sprintf("%s\t[WORKER %v] %v SEEDS %s ...", time.Now().String(), wid, len(seeds), seeds[0].ToString())
						}
					}
					corpus.GUpdateSeed(seeds)
				}()
			}
			seeds := seed.RandomSeed(op_st)
			// if debug {
			// 	if len(seeds) != 0 {
			//		cfg.LogCh <- fmt.Sprintf("%s\t[WORKER %v] %v SEEDS %s ...", time.Now().String(), wid, len(seeds), seeds[0].ToString())
			//		}
			// }
			corpus.GUpdateSeed(seeds)
		}
		ok := fncov.Merge(cov)
		if (init && ok) || !cfg.UseGuide || (inputc != "empty chain" && coveredinput.Len() != 0 && !ctx.timeout) {
			corpus.IncSchedCnt(schedres)
			if info {
				cfg.LogCh <- fmt.Sprintf("%s\t[WORKER %v] NEW score: [%v/%v] Input:%s", time.Now().String(), wid, score, curmax, schedres)
			}
			fncov.UpdateR(schedcov)
			quit = cfg.MaxQuit
			if init && ok { // init can get more coverage, do init instead of mutation
				cfg.InitTurnCnt = cfg.InitTurnCnt * 2
				if cfg.InitTurnCnt > 100 {
					cfg.InitTurnCnt = 100
				}
			}
			go func() { // do mutation in a single routine, check the concurrency safety of corpus
				var mu Mutator
				mu = Mutator{Cov: fncov}
				var ncs []*Chain
				var hts map[uint64]map[uint64]struct{}
				if cfg.UseFeedBack {
					ncs, hts = mu.mutate(coveredinput, energy)
				} else {
					ncs, hts = mu.random(ctx.In.c, 100)
				}
				if debug {
					cfg.LogCh <- fmt.Sprintf("%s\t[WORKER %v] MUTATE %s", time.Now().String(), wid, coveredinput.ToString())
				}
				corpus.Update(ncs, hts) // concurrency safe
			}()
		} else {
			quit -= 1
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
}
