package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// getInstPath 查找 inst 工具路径，优先在可执行文件同目录下查找，
// 失败时回退到 ./bin/inst（适用于 go run 场景）
func getInstPath() string {
	// 先尝试相对于可执行文件的路径
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		instPath := filepath.Join(exeDir, "inst")
		if _, err := os.Stat(instPath); err == nil {
			return instPath
		}
		// Windows 平台也检查 .exe 后缀
		if _, err := os.Stat(instPath + ".exe"); err == nil {
			return instPath + ".exe"
		}
	}
	// 回退：相对于当前工作目录
	return filepath.Join("bin", "inst")
}

func Inst(paths []string, check_pos string, granularity string) {
	// 检查是否有文件需要处理
	if len(paths) == 0 {
		fmt.Println("No Go files found in the specified path")
		return
	}

	resCh := make(chan string, len(paths))
	toolpath := getInstPath()
	// ✅ 新增：限制最大并发数为 16
	maxWorkers := 16
	limit := make(chan struct{}, maxWorkers)

	// 记录成功和失败的文件
	successCount := 0
	failCount := 0
	var failedFiles []string

	dowork := func(path string) {
		command := exec.Command(toolpath, "--file", path, "--checkpos", check_pos, "--granularity", granularity)
		var out, out2 bytes.Buffer
		command.Stdout = &out
		command.Stderr = &out2
		err := command.Run()
		var result string
		if err == nil {
			result = fmt.Sprintf("Handle\t%s OK", path)
		} else {
			result = fmt.Sprintf("Handle\t%s FAIL", path)
		}
		<-limit // 写入结果前先释放槽位，避免分发器死锁
		resCh <- result
	}

	all := len(paths)
	go func() {
		for _, p := range paths {
			limit <- struct{}{}
			go dowork(p)
		}
	}()

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
				fmt.Println("============================")
				return
			}
			//default:
		}
	}
}
