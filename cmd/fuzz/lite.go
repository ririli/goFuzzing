package main

import (
	"fmt"
	"toolkit/cmd"
	"toolkit/pkg/fuzzer"
)

func Lite(bin, fn string, llevel string, timeout, recovertimeout, fuzztime int, maxworker int, granularity string) {
	resCh := make(chan string, 100)
	logCh := make(chan string, 100)
	//bugset := bug.NewBugSet()
	nolimit := make(chan struct{})
	close(nolimit)
	dowork := func(bin string, fn string) {
		m := fuzzer.Monitor{}

		cfg := fuzzer.NewConfig(bin, fn, logCh, "default")
		cfg.LogLevel = llevel
		cfg.TimeOut = timeout
		cfg.RecoverTimeOut = recovertimeout
		cfg.MaxFuzzTime = fuzztime // 0 = 不限时，直传（lite 无默认兜底，与 timeout 的 0 语义不同）
		cfg.MaxWorker = maxworker
		cfg.Granularity = fuzzer.ParseGranularity(granularity)

		ok, detail := m.Start(cfg, nolimit)
		var res string
		if ok {
			res = fmt.Sprintf("%s\tFAIL\t%s\n", fn, detail[1])
		} else {
			res = fmt.Sprintf("%s\tPASS\n", fn)
		}
		resCh <- res
	}
	fmt.Printf("[FUZZER] Start %s\n", bin)
	var cnt, total int
	if fn != "" {
		for i := 0; i < maxworker; i++ {
			go dowork(bin, fn)
		}
		total = maxworker
	} else {
		tests := cmd.ListTests(bin)
		for _, test := range tests {
			fmt.Printf("[WORKER] Start %s\n", test)
			go dowork(bin, test)
		}
		total = len(tests)
	}
	for {
		select {
		case v := <-resCh:
			fmt.Printf("[%v/%v]\t%s", cnt+1, total, v)
			cnt += 1
			if cnt == total {
				return
			}
		case v := <-logCh:
			fmt.Printf("[WORKER] %s\n", v)
		default:
		}
	}
}
