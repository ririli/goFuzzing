package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"toolkit/pkg/inst"
	"toolkit/pkg/utils/gofmt"
)

func ListFiles(d string, f func(s string) bool) []string {
	var files []string

	err := filepath.Walk(d, func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return err
		}
		if info.IsDir() && info.Name() == "vendor" {
			return filepath.SkipDir
		}
		if f(path) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return files
}

// ListTests 返回测试用例列表
func ListTests(bin string) []string {
	res := make([]string, 0)
	fmt.Println("bin = ", bin)
	command := exec.Command(bin, "-test.list", "_1")
	var out bytes.Buffer
	command.Stdout = &out
	err := command.Run()
	if err == nil {
		outstr := out.String()
		ss := strings.Split(outstr, "\n")
		for _, s := range ss {
			if strings.HasPrefix(s, "Test") && strings.HasSuffix(s, "_1") {
				res = append(res, s)
			}
		}
	}
	return res
}

// isGeneratedFile 检查文件是否为 Go generated 文件
// Go 惯例：generated 文件第一行为 "// Code generated <cmd> DO NOT EDIT."
func isGeneratedFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	var buf [256]byte
	n, err := f.Read(buf[:])
	if err != nil || n == 0 {
		return false
	}

	firstLine := strings.SplitN(string(buf[:n]), "\n", 2)[0]
	return strings.HasPrefix(firstLine, "// Code generated") &&
		strings.Contains(firstLine, "DO NOT EDIT")
}

func HandleSrcFile(src string, reg *inst.PassRegistry, passes []string) error {
	if isGeneratedFile(src) {
		log.Printf("skip generated file: %s\n", src)
		return nil
	}

	iCtx, err := inst.NewInstContext(src)
	if err != nil {
		return err
	}

	err = inst.Run(iCtx, reg, passes)
	if err != nil {
		return err
	}

	var dst string
	if Opts.Out != "" {
		dst = Opts.Out
	} else {
		// 就地导出 AST
		dst = iCtx.File

	}
	err = inst.DumpAstFile(iCtx.FS, iCtx.AstFile, dst)
	if err != nil {
		return err
	}

	// 检查输出是否有效，出错则回滚
	if gofmt.HasSyntaxError(dst) {
		// 直接丢弃插桩结果，
		// 并将文件内容还原为原始版本。
		err = ioutil.WriteFile(dst, iCtx.OriginalContent, 0777)
		if err != nil {
			log.Panicf("failed to recover file '%s'", dst)
		}
		log.Printf("recovered '%s' from syntax error\n", dst)
	}

	return nil
}
