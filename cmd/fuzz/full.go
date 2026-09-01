package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
	"toolkit/cmd"
	"toolkit/pkg/bug"
	"toolkit/pkg/fuzzer"
)

func Full(path string, llevel string, feature string, maxworker int, timeout, rtimeout, fuzztime int, granularity string, outDir string) {
	if timeout == 0 {
		timeout = 30
	}
	if rtimeout == 0 {
		rtimeout = 200
	}
	if outDir == "" {
		outDir = "gopieRes"
	}
	// fuzztime 不做 0→默认 归一化：0 表示"不限时"（与上方 timeout/rtimeout 的 0→默认 兜底语义相反，勿混淆）
	startTime := time.Now()
	// 跨测试共享的 bug 集合：fuzzing 过程中收集 panic/data race 事件，
	// 结束后聚合追加到 outDir 下的 allpanic.txt / alldatarace.txt
	sharedBugs := bug.NewSet()
	defer func() {
		if err := dumpBugReports(sharedBugs, outDir); err != nil {
			fmt.Printf("[Fuzzer] dump bug reports failed: %v\n", err)
		} else {
			fmt.Printf("[Fuzzer] bug reports appended to %s (allpanic.txt / alldatarace.txt)\n", outDir)
		}
	}()
	resCh := make(chan string, 100000)
	logCh := make(chan string, 100000)
	// 并发控制
	max := 12
	if maxworker != 0 {
		max = maxworker
	}
	limit := make(chan struct{}, max*2)
	for i := 0; i < max; i++ {
		limit <- struct{}{}
	}

	// 二进制文件对应的测试函数
	bin2tests := make(map[string][]string)

	//bugset := bug.NewBugSet()

	bins := cmd.ListFiles(path, func(s string) bool {
		return true
	})

	// 将测试与 visitor 绑定到各二进制文件
	total := 0
	for _, bin := range bins {
		tests := cmd.ListTests(bin)
		bin2tests[bin] = tests
		total += len(tests)
	}

	go func() {
		fmt.Println("----len bin2tests=", len(bin2tests)) // 测试文件的数量
		for bin, tests := range bin2tests {
			fmt.Println("--len tests=", len(tests)) // 单个文件测试函数的数量
			for _, test := range tests {
				cfg := fuzzer.DefaultConfig() //fuzzing 配置
				// 共享 bugset
				//cfg.BugSet = bugset
				cfg.Bin = bin
				cfg.Fn = test
				cfg.MaxWorker = 4
				cfg.TimeOut = timeout
				cfg.RecoverTimeOut = rtimeout
				cfg.MaxFuzzTime = fuzztime
				cfg.LogCh = logCh
				cfg.MaxQuit = 200 // 退出循环次数
				cfg.MaxExecution = 250
				cfg.LogLevel = llevel
				cfg.Granularity = fuzzer.ParseGranularity(granularity)
				cfg.SharedBugs = sharedBugs
				if feature == "mu" {
					cfg.UseMutate = false
				}
				if feature == "fb" {
					cfg.UseFeedBack = false
				}

				<-limit
				go func(cfg *fuzzer.Config) {
					defer func() {
						limit <- struct{}{}
					}()
					m := &fuzzer.Monitor{}
					ok, detail := m.Start(cfg, limit)
					var res string
					if ok {
						res = fmt.Sprintf("%s\tFAIL\t%s\n", cfg.Fn, detail[1])
					} else {
						res = fmt.Sprintf("%s\tPASS\n", cfg.Fn)
					}
					resCh <- res
				}(cfg)
			}
		}
	}()

	defer func() {
		fmt.Printf("%v [Fuzzer] Finish, elapsed: %.3fs\n", time.Now().String(), time.Since(startTime).Seconds())
	}()
	if total == 0 {
		fmt.Println("no tests found")
		return
	}
	cnt := 0
	for {
		select {
		case v := <-resCh:
			fmt.Printf("%v [%v/%v]\t%s", time.Now().String(), cnt+1, total, v)
			cnt++
			if cnt == total {
				return
			}
		case v := <-logCh:
			fmt.Printf("%v [WORKER] %s\n", time.Now().String(), v)
		default:
		}
	}
}

// dumpBugReports 将共享 bug 集合中的完整报告按种类追加写入 outDir：
//   - panic     → allpanic.txt
//   - data race → alldatarace.txt
//
// 采用追加模式：run_full 脚本按二进制逐个调用 fuzz，各轮的报告跨进程
// 累积到同一对文件；脚本在整轮实验开始时负责清理旧文件。
// fatal/hang/test_order_panic 等其他种类不写入，仍可在全量 stdout 日志中查看。
func dumpBugReports(bugs *bug.Set, outDir string) error {
	if bugs == nil {
		return nil
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	var panics, races []bug.Record
	for _, rec := range bugs.Snapshot() {
		switch rec.Kind {
		case bug.KindPanic:
			panics = append(panics, rec)
		case bug.KindDataRace:
			races = append(races, rec)
		}
	}

	if err := appendBugRecords(filepath.Join(outDir, "allpanic.txt"), panics); err != nil {
		return err
	}
	return appendBugRecords(filepath.Join(outDir, "alldatarace.txt"), races)
}

// appendBugRecords 将记录追加到指定文件；无记录时仅确保文件存在。
func appendBugRecords(path string, records []bug.Record) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for i, rec := range records {
		fmt.Fprintf(f, "======== %s #%d ========\n", rec.Kind, i+1)
		fmt.Fprintf(f, "bin=%s test=%s count=%d associated=%d signature=%s\n",
			rec.Last.Bin, rec.Last.Fn, rec.Count, rec.AssociatedCount, rec.Signature)
		fmt.Fprintf(f, "%s\n\n", rec.Report)
	}
	return nil
}
