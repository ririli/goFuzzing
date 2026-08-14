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
	fmt.Printf("[DEBUG] goPath = %s\n", goPath)

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
		replacedDir := replacer.Replace(dir)
		// 如果 path 为 "."，replacedDir 也是 "."，filepath.Join 会把它清掉导致路径错误
		// 此时用绝对路径的目录名作为二进制名称
		if replacedDir == "." {
			absDir, _ := filepath.Abs(dir)
			replacedDir = filepath.Base(absDir)
		}
		opath := filepath.Join(workpath, outputDir, replacedDir)
		// 在 Windows 上 go 工具会生成 .exe 文件。让输出文件名显式带上
		// 该后缀，以便下游尝试执行二进制文件的调用方能够可靠地找到它。
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

	// 统计成功和失败
	successCount := 0
	failCount := 0
	var failedDirs []string

	all := len(dirs)
	if all == 0 {
		fmt.Println("No directories to process")
		return
	}
	for {
		select {
		case v := <-resCh:
			fmt.Printf("[%v/%v]\t%s\n", len(dirs)-all+1, len(dirs), v)

			if strings.HasSuffix(v, "OK") {
				successCount++
			} else {
				failCount++
				// 提取失败目录路径: "Handle\t<dir> FAIL" -> <dir>
				parts := strings.TrimPrefix(v, "Handle\t")
				parts = strings.TrimSuffix(parts, " FAIL")
				failedDirs = append(failedDirs, parts)
			}

			all -= 1
			if all == 0 {
				// 输出统计信息
				fmt.Println("\n========== 编译统计 ==========")
				fmt.Printf("总目录数: %d\n", len(dirs))
				fmt.Printf("成功: %d\n", successCount)
				fmt.Printf("失败: %d\n", failCount)
				if len(dirs) > 0 {
					successRate := float64(successCount) / float64(len(dirs)) * 100
					fmt.Printf("成功率: %.2f%%\n", successRate)
				}
				if failCount > 0 {
					fmt.Println("\n失败目录列表:")
					for i, dir := range failedDirs {
						fmt.Printf("  %d. %s\n", i+1, dir)
					}
				}
				fmt.Println("==============================")
				fmt.Printf("\n共找到 %v 个测试:\n", len(tests))
				for _, test := range tests {
					fmt.Println(test)
				}
				return
			}
		default:
		}
	}
}
