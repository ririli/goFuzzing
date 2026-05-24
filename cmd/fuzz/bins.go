package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"toolkit/cmd"
)

func Bins(paths []string, outputDir string) {
	resCh := make(chan string, 100)
	tests := make([]string, 0)
	var mu sync.Mutex
	goPath, err := exec.LookPath("go")
	if err != nil {
		fmt.Println("Error: go compiler not found in PATH")
		return
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
	outputPath := filepath.Join(workpath, outputDir)
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
		replacer := strings.NewReplacer(":", "_", "\\", "_", "/", "_")
		opath := filepath.Join(workpath, outputDir, replacer.Replace(dir))
		// On Windows the go tool will produce a .exe file. Make the output
		// filename explicitly include the suffix so downstream callers that
		// try to execute the binary can find it reliably.
		if runtime.GOOS == "windows" {
			opath = opath + ".exe"
		}
		command := exec.Command(goPath, "test", "-race", "-o", opath, "-c", ".")
		command.Dir = dir
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
		dir := filepath.Dir(path)
		if _, ok := m[dir]; !ok && dir != "" {
			dirs = append(dirs, dir)
			m[dir] = struct{}{}
		}
	}
	for _, dir := range dirs {
		go dowork(dir)
	}

	all := len(dirs)
	if all == 0 {
		fmt.Println("No directories to process")
		return
	}
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
