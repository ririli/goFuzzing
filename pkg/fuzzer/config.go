package fuzzer

import (
	"os"
	"strings"
)

// GranularityMode 定义 fuzzing 颗粒度级别。
type GranularityMode string

const (
	// ModeGoroutine 调度 goroutine 对（go 语句级别）。
	ModeGoroutine GranularityMode = "goroutine"
	// ModeFunction 调度函数调用对（函数入口级别）。
	ModeFunction GranularityMode = "function"
)

// ParseGranularity 从字符串解析颗粒度模式（不区分大小写），非法值默认返回 goroutine。
func ParseGranularity(s string) GranularityMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "function", "func":
		return ModeFunction
	default:
		return ModeGoroutine
	}
}

// GetGranularityFromEnv 从环境变量 FUZZ_MODE 读取颗粒度设置。
// 未设置时默认返回 goroutine 模式。
func GetGranularityFromEnv() GranularityMode {
	mode := os.Getenv("FUZZ_MODE")
	return ParseGranularity(mode)
}

type Config struct {
	Bin string
	Fn  string

	MaxWorker    int
	MaxExecution int
	SingleCrash  bool

	LogLevel string
	LogCh    chan string

	UseFeedBack     bool
	UseCoveredSched bool
	UseStates       bool
	UseAnalysis     bool
	UseMutate       bool
	UseGuide        bool

	TimeOut         int
	RecoverTimeOut  int
	InitTurnCnt     int
	MaxQuit         int
	MaxPreExecRound int             // 预执行轮次上限
	GortPhase       uint32          // 0=预执行种子收集, 1=fuzzing阶段 (CorpusGort/CorpusFunc写入, CorpusOp读取)
	Granularity     GranularityMode // "goroutine" 或 "function"——调度哪种并发对

	//BugSet *bug.BugSet
}

func DefaultConfig() *Config {
	c := &Config{
		MaxWorker:       5,
		MaxExecution:    10000,
		SingleCrash:     false,
		LogLevel:        "normal",
		UseFeedBack:     true,
		UseStates:       true,
		UseCoveredSched: true,
		UseAnalysis:     true,
		UseMutate:       true,
		UseGuide:        true,
		TimeOut:         30,
		RecoverTimeOut:  100,
		InitTurnCnt:     100,
		MaxQuit:         500,
		MaxPreExecRound: 30,
		Granularity:     ModeGoroutine,
	}
	return c
}

func GokerConfig() *Config {
	c := &Config{
		MaxWorker:       5,
		MaxExecution:    100000,
		SingleCrash:     true,
		LogLevel:        "normal",
		UseFeedBack:     true,
		UseStates:       true,
		UseCoveredSched: true,
		UseAnalysis:     true,
		UseMutate:       true,
		UseGuide:        true,
		TimeOut:         30,
		RecoverTimeOut:  100,
		InitTurnCnt:     0,
		MaxQuit:         10000,
	}
	return c
}

func NewConfig(bin, fn string, logCh chan string, typ string) *Config {
	var c *Config
	switch typ {
	case "goker":
		c = GokerConfig()
	default:
		c = DefaultConfig()
	}
	c.Bin = bin
	c.Fn = fn
	c.LogCh = logCh
	//	c.BugSet = bugset
	return c
}
