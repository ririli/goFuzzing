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
	//log.Println(http.ListenAndServe(":6060", nil))
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
	corpusGort = NewCorpusGort()
	var corpusOp *CorpusOp
	corpusOp = NewCorpusOp()
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
			// OP对调度已禁用，仅使用goroutine级别调度
			e := Executor{}
			in := Input{
				gortPair: gortPair,
				cmd:      cfg.Bin,
				args:     []string{"-test.v", "-test.run", cfg.Fn},
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
	fmt.Println("m.max=", m.max) // 最大运行次数上限
	for {
		fmt.Println("m.etimes=", m.etimes) // 已执行的轮次
		if m.etimes > m.max {
			close(cancel)
			return false, []string{}
		}
		ctx := <-ch // 接收worker的执行结果
		atomic.AddInt32(&m.etimes, 1)
		var inputc string

		inputc = "empty chain"

		if debug {
			cfg.LogCh <- fmt.Sprintf("%s\t[WORKER %v] Input: %s", time.Now().String(), wid, inputc)
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
				//close(cancel)
				//return true, []string{inputc, "DATA RACE", ""}
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
			//close(cancel)
			//return true, []string{inputc, "DATA RACE", raceReport}
		}
		// 输出台收集信息
		// stderr → 种子信息（预执行和 fuzzing 全程收集）
		pair_st, opInfos, err := feedback.ParseGortPairs(ctx.Out.Trace)
		if err == nil {
			if len(pair_st) > 0 {
				corpusGort.AddPair(pair_st)
			}
			if len(opInfos) > 0 {
				corpusOp.Add(opInfos)
				// Rule 1: 共享对象访问推测 — 同对象且不同gid → SusConPairs
				if sharedPairs := inferSharedObjectPairs(opInfos); len(sharedPairs) > 0 {
					corpusGort.AddPair(sharedPairs)
				}
			}
		}

		// 预执行阶段判断
		if atomic.LoadUint32(&corpusGort.done) == 0 {
			if corpusGort.TryEndPreExec(cfg.MaxPreExecRound) {
				fmt.Printf("[PRESTAGE] Pre-execution finished, total pairs: cover=%d\n",
					len(corpusGort.CoveredConPairs))
			}
			//continue
		}

		// --- fuzzing 阶段 ---
		// stdout → 调度有效性信号
		funcSignals, _ := feedback.ParseSignals(ctx.Out.O)
		if cfg.UseMutate {
			if len(funcSignals) > 0 {
				corpusGort.ApplySignals(funcSignals)
			}
			// OP信号调度已禁用，仅使用goroutine级别调度
		}

		// todo 有价值就继续fuzzing，不减quit
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

// inferSharedObjectPairs Rule 1: 共享对象访问推测
// 从 [FB] 操作日志中提取操作了同一channel/WG对象的不同goroutine，
// 生成 SusConPairs（置信度0.8），用于后续的goroutine级别fuzzing调度
func inferSharedObjectPairs(opInfos []*feedback.OpInfo) []*feedback.GortPairInfo {
	// 按 (ObjKind, ObjAddr) 分组
	type objKey struct {
		kind string
		addr uint64
	}
	byObject := make(map[objKey]map[uint64]bool) // objKey -> set of gids
	for _, op := range opInfos {
		if op == nil || op.Gid == 0 {
			continue
		}
		key := objKey{kind: string(op.ObjKind), addr: op.ObjAddr}
		if byObject[key] == nil {
			byObject[key] = make(map[uint64]bool)
		}
		byObject[key][op.Gid] = true
	}

	// 对每个共享对象，生成不同goroutine之间的配对
	var pairs []*feedback.GortPairInfo
	seen := make(map[string]bool)
	for _, gids := range byObject {
		gidList := make([]uint64, 0, len(gids))
		for gid := range gids {
			gidList = append(gidList, gid)
		}
		// 需要至少2个不同的gid
		for i := 0; i < len(gidList); i++ {
			for j := i + 1; j < len(gidList); j++ {
				gid1, gid2 := gidList[i], gidList[j]
				if gid1 > gid2 {
					gid1, gid2 = gid2, gid1
				}
				key := fmt.Sprintf("%d-%d", gid1, gid2)
				if seen[key] {
					continue
				}
				seen[key] = true
				pairs = append(pairs, &feedback.GortPairInfo{
					Gid1:       gid1,
					Gid2:       gid2,
					Confidence: 0.8,
					SourceType: "inferred_shared_obj",
					IsObserved: false,
				})
			}
		}
	}
	return pairs
}
