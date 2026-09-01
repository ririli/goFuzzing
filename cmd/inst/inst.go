package main

import (
	"log"
	"os"
	"strings"
	"toolkit/cmd"
	"toolkit/pkg/inst"
	passes "toolkit/pkg/inst/passes"
)

func main() {
	cmd.ParseFlags()
	// 归一化颗粒度（大小写不敏感，支持 func 别名），后续一律使用归一化后的值
	granularity := cmd.NormalizeGranularity(cmd.Opts.Granularity)
	isFunc := granularity == "function"

	if cmd.Opts.Dir != "" {
		files := cmd.ListFiles(cmd.Opts.Dir, func(s string) bool {
			return strings.Contains(s, ".go")
		})

		failed := 0
		for _, file := range files {
			reg := inst.NewPassRegistry()

			if isFunc {
				reg.Register("func", func() inst.InstPass { return &passes.FunctionPass{} })
			} else {
				reg.Register("gort", func() inst.InstPass { return &passes.GoroutinePass{} })
			}
			reg.Register("channel", func() inst.InstPass { return &passes.ChRecPass{} })
			reg.Register("select", func() inst.InstPass { return &passes.SelectPass{} })
			reg.Register("waitgroup", func() inst.InstPass { return &passes.WgPass{} })
			reg.Register("test", func() inst.InstPass { return &passes.TestPass{Pos: cmd.Opts.Pos, Granularity: granularity} })

			err := cmd.HandleSrcFile(file, reg, reg.ListOfPassNames())
			log.Println("Inst " + file)
			if err != nil {
				log.Printf("error %v", err.Error())
				failed++
			}
		}
		// 有失败文件时以非零退出码结束，避免调度方误判全部成功
		if failed > 0 {
			os.Exit(1)
		}
	} else {
		if cmd.Opts.File == "" {
			log.Fatalf("Need source file")
		}
		reg := inst.NewPassRegistry()

		if isFunc {
			reg.Register("func", func() inst.InstPass { return &passes.FunctionPass{} })
		} else {
			reg.Register("gort", func() inst.InstPass { return &passes.GoroutinePass{} })
		}
		reg.Register("channel", func() inst.InstPass { return &passes.ChRecPass{} })
		reg.Register("select", func() inst.InstPass { return &passes.SelectPass{} })
		reg.Register("waitgroup", func() inst.InstPass { return &passes.WgPass{} })
		reg.Register("test", func() inst.InstPass { return &passes.TestPass{Pos: cmd.Opts.Pos, Granularity: granularity} })

		if err := cmd.HandleSrcFile(cmd.Opts.File, reg, reg.ListOfPassNames()); err != nil {
			log.Printf("error %v", err.Error())
			os.Exit(1)
		}
	}
}
