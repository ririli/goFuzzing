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
		fmt.Printf("[FUZZER] %s elapsed: %v, etimes=%d\n", cfg.Fn, time.Since(startTime).Round(time.Millisecond), atomic.LoadInt32(&m.etimes))
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
	// todo 实现输入和输出
	var corpusPair *CorpusPair
	corpusPair = NewCorpusPair()

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

			tryPair := corpusPair.Get()

			e := Executor{}
			in := Input{
				tryPair: tryPair,
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
	cfg.MaxWorker = 2
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
		// 输出台收集信息
		pair_st, err := feedback.ParseStdPairs(ctx.Out.Trace)
		//pair_st, err := feedback.ParseStdPairs("[SUSPECT] 987842478097,987842478084|gopie/testdata/gobench/nonblocking/grpc/1748/grpc1748_test.go:184,gopie/testdata/gobench/nonblocking/grpc/1748/grpc1748_test.go (Test):171|0.50|inferred_child1;\n]")
		if err == nil && len(pair_st) > 0 {

			corpusPair.AddPair(pair_st)
		}

		// if len(schedcov) != 0 && cfg.UseCoveredSched {
		//	score += (len(schedcov) / (ctx.In.c.Len())) * len(schedcov) * 10
		// }

		init := atomic.LoadUint32(&m.doinit) == 1
		if init && atomic.LoadInt32(&m.etimes) > int32(cfg.InitTurnCnt) {
			atomic.StoreUint32(&m.doinit, 0)
			if cfg.UseMutate {
				fmt.Printf("[MUTATE] SWITCH TO MUTATION MODE, CURRENT INITCNT %v\n", cfg.InitTurnCnt)
			}
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
