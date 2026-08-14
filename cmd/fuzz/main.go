package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"toolkit/cmd"

	"github.com/jessevdk/go-flags"
)

var opts struct {
	T           string `long:"timeout" description:"Instrument single go source file"`
	RT          string `long:"recovertimeout" description:"Output instrumented golang source file to the given file. Only allow when instrumenting single golang source file"`
	PATH        string `long:"path" description:"path"`
	TASK        string `long:"task" description:"task"`
	LL          string `long:"llevel" description:"log level [info, debug, normal]"`
	MaxWoker    string `long:"max" description:"max workers"`
	Fn          string `long:"func" description:"function"`
	Feature     string `long:"feature" description:"[full, fb (without feedback), mu (without mutation)]"`
	LeakCheck   string `long:"check" description:"the position of leakcheck [inside, outside]"`
	Output      string `long:"output" short:"o" description:"output directory for binary files"`
	Granularity string `long:"granularity" description:"fuzzing granularity [goroutine, function]. Overrides FUZZ_MODE env var."`
}

func ParseFlags() {

	if _, err := flags.Parse(&opts); err != nil {
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

// resolveGranularity 解析颗粒度：CLI 标志优先，否则从 FUZZ_MODE 环境变量读取，
// 最终统一经 cmd.NormalizeGranularity 归一化（大小写不敏感，支持 func 别名，默认 goroutine）。
func resolveGranularity() string {
	if opts.Granularity != "" {
		return cmd.NormalizeGranularity(opts.Granularity)
	}
	return cmd.NormalizeGranularity(os.Getenv("FUZZ_MODE"))
}

func main() {
	ParseFlags()
	granularity := resolveGranularity()
	fmt.Printf("[FUZZ] Granularity mode: %s\n", granularity)

	switch opts.TASK {
	case "lite":
		var timeout, rtimeout int64
		var maxworker int
		if opts.RT != "" {
			rtimeout, _ = strconv.ParseInt(opts.RT, 10, 32)
		}
		if opts.T != "" {
			timeout, _ = strconv.ParseInt(opts.T, 10, 32)
		}
		if opts.MaxWoker != "" {
			max, _ := strconv.ParseInt(opts.MaxWoker, 10, 32)
			maxworker = int(max)
		}
		Lite(opts.PATH, opts.Fn, opts.LL, int(timeout), int(rtimeout), maxworker, granularity)
	case "full":
		var timeout, rtimeout int64
		if opts.RT != "" {
			rtimeout, _ = strconv.ParseInt(opts.RT, 10, 32)
		}
		if opts.T != "" {
			timeout, _ = strconv.ParseInt(opts.T, 10, 32)
		}
		var maxworker int
		if opts.MaxWoker != "" {
			max, _ := strconv.ParseInt(opts.MaxWoker, 10, 32)
			maxworker = int(max)
		}
		Full(opts.PATH, opts.LL, opts.Feature, maxworker, int(timeout), int(rtimeout), granularity)
	case "inst":
		paths := cmd.ListFiles(opts.PATH, func(s string) bool {
			return strings.HasSuffix(s, ".go")
		})
		pos := "outside"
		if opts.LeakCheck != "" {
			pos = opts.LeakCheck
		}
		Inst(paths, pos, granularity)
	case "bins":
		paths := cmd.ListFiles(opts.PATH, func(s string) bool {
			return strings.HasSuffix(s, "_test.go")
		})
		Bins(paths, opts.Output)
	default:
		fmt.Println("error argument" + " " + opts.TASK)
	}
}
