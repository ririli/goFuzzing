package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
)

const (
	localPath = "D:\\Program Files\\goProjects\\src\\gopie\\bin\\inst.exe"
	linuxPath = "./bin/inst"
)

func Inst(paths []string, check_pos string) {
	// 检查是否有文件需要处理
	if len(paths) == 0 {
		fmt.Println("No Go files found in the specified path")
		return
	}

	resCh := make(chan string, 100)
	var toolpath = localPath
	if runtime.GOOS == "linux" {
		toolpath = linuxPath
	}
	// ✅ 新增：限制最大并发数为 16
	maxWorkers := 16
	limit := make(chan struct{}, maxWorkers)

	// 记录成功和失败的文件
	successCount := 0
	failCount := 0
	var failedFiles []string

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

			// 统计成功和失败
			if len(v) >= 2 && v[len(v)-2:] == "OK" {
				successCount++
			} else {
				failCount++
				// 提取失败的文件路径
				if len(v) > 7 { // "Handle\t" 的长度是 7
					failedFiles = append(failedFiles, v[7:len(v)-5]) // 去掉 "Handle\t" 和 " FAIL"
				}
			}

			all--
			if all == 0 {
				// 输出统计信息
				fmt.Println("\n========== 插桩统计 ==========")
				fmt.Printf("总文件数: %d\n", len(paths))
				fmt.Printf("成功: %d\n", successCount)
				fmt.Printf("失败: %d\n", failCount)
				if len(paths) > 0 {
					successRate := float64(successCount) / float64(len(paths)) * 100
					fmt.Printf("成功率: %.2f%%\n", successRate)
				}

				if failCount > 0 {
					fmt.Println("\n失败文件列表:")
					for i, file := range failedFiles {
						fmt.Printf("  %d. %s\n", i+1, file)
					}
				}
				fmt.Println("============================\n")
				return
			}
			//default:
		}
	}
}
