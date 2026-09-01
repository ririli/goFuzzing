package cmd

import (
	"os"
	"strings"

	flags "github.com/jessevdk/go-flags"
)

var Opts struct {
	File        string `long:"file" description:"Instrument single go source file"`
	Out         string `long:"out" description:"Output instrumented golang source file to the given file. Only allow when instrumenting single golang source file"`
	Dir         string `long:"dir" description:"Instrument all go source file under this dir"`
	OnlyGoleak  string `long:"onlygoleak" description:"only goleak inst"`
	Pos         string `long:"checkpos" description:"leak check position"`
	Granularity string `long:"granularity" description:"fuzzing granularity [goroutine, function]" default:"goroutine"`
}

// NormalizeGranularity 将颗粒度参数归一化为 "goroutine" 或 "function"：
// 大小写不敏感，支持 function/func 别名，非法值回退为 goroutine。
// 所有 CLI 入口（inst/full/lite）必须先经此函数归一化，
// 禁止对颗粒度字符串直接做精确比较或类型强转；
// 语义必须与 fuzzer.ParseGranularity 保持一致。
func NormalizeGranularity(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "function", "func":
		return "function"
	default:
		return "goroutine"
	}
}

func ParseFlags() {
	if _, err := flags.Parse(&Opts); err != nil {
		switch flagsErr := err.(type) {
		case flags.ErrorType:
			if flagsErr == flags.ErrHelp {
				os.Exit(0)
			}
			os.Exit(1)
		default:
			os.Exit(1)
		}
	}
}
