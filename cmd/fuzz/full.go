package main

import (
	"fmt"
	"time"
	"toolkit/cmd"
	"toolkit/pkg/fuzzer"
)

func Full(path string, llevel string, feature string, maxworker int, timeout, rtimeout int, granularity string) {
	if timeout == 0 {
		timeout = 30
	}
	if rtimeout == 0 {
		rtimeout = 200
	}
	startTime := time.Now()
	resCh := make(chan string, 100000)
	logCh := make(chan string, 100000)
	// control
	max := 24
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

	// bind tests and visitor to bins
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
				cfg := fuzzer.DefaultConfig() //fuzzing Config
				// shared bugset
				//cfg.BugSet = bugset
				cfg.Bin = bin
				cfg.Fn = test
				cfg.MaxWorker = 4
				cfg.TimeOut = timeout
				cfg.RecoverTimeOut = rtimeout
				cfg.LogCh = logCh
				cfg.MaxQuit = 200 // 推出循环次数
				cfg.MaxExecution = 250
				cfg.LogLevel = llevel
				cfg.Granularity = fuzzer.GranularityMode(granularity)
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
