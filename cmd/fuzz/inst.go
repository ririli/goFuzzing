package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
)

const (
	localPath = "D:\\Program Files\\goProjects\\src\\gopie\\bin\\inst.exe"
	linuxPath = ""
)

func Inst(paths []string, check_pos string) {
	resCh := make(chan string, 100)
	var toolpath = localPath
	if runtime.GOOS == "linux" {
		toolpath = linuxPath
	}
	// ✅ 新增：限制最大并发数为 16
	maxWorkers := 16
	limit := make(chan struct{}, maxWorkers)
	dowork := func(path string) {
		defer func() {
			<-limit
		}()
		command := exec.Command(toolpath, "--file", path, "--checkpos", check_pos) // 执行inst二进制文件
		var out, out2 bytes.Buffer
		command.Stdout = &out
		command.Stderr = &out2
		err := command.Run()
		if err == nil {
			resCh <- fmt.Sprintf("Handle\t%s OK", path)
		} else {
			resCh <- fmt.Sprintf("Handle\t%s FAIL", path)
		}
	}

	all := len(paths)
	for _, p := range paths {
		limit <- struct{}{}
		go dowork(p)
	}

	for {
		select {
		case v := <-resCh:
			fmt.Printf("[%v/%v]\t%s\n", len(paths)-all+1, len(paths), v)
			all--
			if all == 0 {
				return
			}
			//default:
		}
	}
}
