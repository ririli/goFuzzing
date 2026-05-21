package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"toolkit/cmd"
)

const (
	localGo = "D:\\Program Files\\GO\\go1.19_patch\\bin\\go.exe"
	linuxGo = "/home/lichang/local/go1.19.1/go/bin/go"
)

func dirname(s string) string {
	if strings.Contains(s, ".go") {
		idx := strings.LastIndex(s, "/")
		if idx != -1 {
			return s[0:idx]
		}
	}
	return ""
}

func Bins(paths []string, outputDir string) {
	resCh := make(chan string, 100)
	tests := make([]string, 0)
	var mu sync.Mutex
	goPath := localGo
	if runtime.GOOS == "linux" {
		goPath = linuxGo
	}

	limit := make(chan struct{}, 32)
	for i := 0; i < 16; i++ {
		limit <- struct{}{}
	}

	workpath, _ := os.Getwd()

	// 如果未指定输出目录，使用默认的 testbins
	if outputDir == "" {
		outputDir = "testbins"
	}

	// 确保输出目录存在，如果不存在则创建
	outputPath := workpath + "/" + outputDir
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		fmt.Printf("Error creating output directory %s: %v\n", outputPath, err)
		return
	}
	fmt.Printf("Output directory: %s\n", outputPath)

	dowork := func(dir string) {
		<-limit
		defer func() {
			limit <- struct{}{}
		}()
		opath := workpath + "/" + outputDir + "/" + strings.Replace(dir, "/", "_", -1)
		c := fmt.Sprintf("cd %s && %s test -race -o %s -c .", dir, goPath, opath)
		command := exec.Command("bash", "-c", c)
		var out, out2 bytes.Buffer
		command.Stdout = &out
		command.Stderr = &out2
		err := command.Run()
		if err == nil {

			t := cmd.ListTests(opath)
			mu.Lock()
			tests = append(tests, t...)
			mu.Unlock()
			resCh <- fmt.Sprintf("Handle\t%s OK", opath)
		} else {
			fmt.Printf("Error compiling %s:\nStdout: %s\nStderr: %s\n", dir, out.String(), out2.String())
			resCh <- fmt.Sprintf("Handle\t%s FAIL", dir)
		}
	}

	m := make(map[string]struct{})
	dirs := make([]string, 0)
	for _, path := range paths {
		dir := dirname(path)
		if _, ok := m[dir]; !ok && dir != "" {
			dirs = append(dirs, dir)
			m[dir] = struct{}{}
		}
	}
	for _, dir := range dirs {
		go dowork(dir)
	}

	all := len(dirs)
	for {
		select {
		case v := <-resCh:
			fmt.Printf("[%v/%v]\t%s\n", len(dirs)-all+1, len(dirs), v)
			all -= 1
			if all == 0 {
				fmt.Printf("Finish, Find %v Tests:\n", len(tests))
				for _, test := range tests {
					fmt.Println(test)
				}
				return
			}
		default:
		}
	}
}
