# GoPie - Golang 并发 Fuzzing 测试工具

## 项目概述

GoPie 是一个基于 Fuzzing 的 Golang 数据竞争检测工具。通过对被测代码进行 **AST 插桩**（AST Instrumentation），在测试执行时收集并发行为信息，构建 goroutine 调用顺序图，并据此引导 Fuzzing 变异策略来系统性地检测数据竞争、panic。当前项目以 goroutine 和操作（channel/WaitGroup）为检测颗粒度，函数颗粒度代码（callstack / SuspiciousPairInfo）已废弃但保留。

## AI 交互准则

- 回答问题时保持客观实事求是，不刻意迎合用户的想法
- 若用户的方案、判断或假设存在明显问题，应明确指出错误并给出纠正建议，而非一味顺从
- 对模糊的需求应先澄清再动手，避免基于错误理解展开工作

## 技术栈

- **语言**: Go 1.19.1
- **模块名**: `toolkit`
- **核心依赖**: `go-flags`（命令行解析）、`testify`（测试框架）、`x/tools`（Go AST 工具集）

## 项目结构

```
gopie/
├── cmd/                          # 命令行入口
│   ├── fuzz/                     # fuzz 主命令（inst / bins / full / lite）
│   │   ├── main.go               # 入口，路由到各子任务
│   │   ├── inst.go               # inst 任务实现
│   │   ├── bins.go               # bins 编译任务实现
│   │   ├── full.go               # full fuzzing 任务实现
│   │   └── lite.go               # lite 简化模式
│   ├── inst/inst.go              # 独立 inst 命令
│   ├── flags.go                  # inst 子命令标志定义
│   └── utils.go                  # 通用工具（文件遍历等）
├── pkg/                          # 核心包
│   ├── inst/                     # AST 插桩引擎
│   │   ├── instctx.go            # 插桩上下文（解析、类型检查、opid 分配）
│   │   ├── registry.go           # Pass 注册中心（Register/GetNewPassInstance）
│   │   ├── run.go                # Pass 执行引擎（Before → Apply → After 生命周期）
│   │   ├── type.go               # InstPass / InstContext 类型定义
│   │   ├── err.go                # 错误类型
│   │   ├── util.go               # 工具函数
│   │   └── passes/               # 各类插桩 Pass 实现
│   │       ├── func.go           # 函数 funcID 分配（不再生成 callstack 调用）
│   │       ├── gort.go           # goroutine Enter/Exit 插桩（go 语句包装）
│   │       ├── channel.go        # channel send/close 插桩（BF + AF 钩子）
│   │       ├── select.go         # select 分支 send 插桩（AF 钩子）
│   │       ├── waitgroup.go      # WaitGroup Add/Done 插桩（BF + AF 钩子）
│   │       ├── global.go         # 全局 ID 映射（pos → id）
│   │       ├── test.go           # 测试函数包装（生成 TestXxx_1）
│   │       └── utils.go          # Pass 公共工具（AST 节点构造、类型检查）
│   ├── fuzzer/                   # Fuzzing 引擎
│   │   ├── config.go             # Config 配置（含 GortPhase 阶段标志）
│   │   ├── corpus_gort.go        # CorpusGort：goroutine 级别种子管理
│   │   ├── corpus_op.go          # CorpusOp：操作级别种子管理
│   │   ├── executor.go           # Executor：测试二进制执行与输出流式处理
│   │   └── monitor.go            # Monitor：主监控循环（预执行 + Fuzzing 两阶段）
│   ├── goroutine/                # 运行时 Goroutine 生命周期追踪与并发对推断
│   │   ├── gort.go               # Enter/Exit 生命周期 hook，CurrentGid，EnterMain/ExitMain
│   │   ├── control.go            # pointControl goroutine 断点同步
│   │   ├── parse.go              # ParseInput 解析调度配置
│   │   ├── overlap.go            # GoroutineTracker / GoroutineRecord 时间追踪
│   │   ├── infer.go              # 并发对推断规则（时间重叠 + 三类推测）+ PrintGoroutinePairs
│   │   └── env.go                # Config 调度配置（barriers/activeMap）
│   ├── feedback/                 # 反馈信号类型与解析
│   │   ├── type.go               # SuspiciousPairInfo / InputPair 类型（函数级，保留未使用）
│   │   ├── signal.go             # CoverageSignal / SignalKind 定义
│   │   ├── format.go             # ParseSignals / ParseGortPairs / ParseStdPairs 信号解析
│   │   ├── type_gort.go          # GortPairInfo / InputGortPair 类型
│   │   └── type_op.go            # OpInfo / OpPair / InputOpPair / MatchOpPair 类型
│   ├── sched/                    # 操作级调度原语（[FB] 日志 + OP 断点同步）
│   │   ├── sched.go              # InstChBF/AF / InstWgBF/AF / InstChSelectAF 钩子
│   │   └── env.go                # Config（preOpMap/waitMap/active）
│   ├── bug/bug.go                # Bug 结果汇总（待实现）
│   └── utils/                    # 通用工具
│       ├── gofmt/fmt.go          # gofmt 格式化
│       └── hash/hash.go          # hash 工具
├── script/                       # 构建 & 运行脚本
├── testdata/                     # 被测项目测试数据
├── testbins/                     # 编译后的测试二进制
├── raceResult/                   # Race 检测结果汇总
├── zgortResult/                  # Goroutine 对结果汇总
├── go.mod / go.sum               # Go 模块定义
└── README.md
```

## 核心架构

### 1. AST 插桩

GoPie 通过 AST Pass 系统对用户源码进行静态插桩。并发行为的追踪和调度完全由插桩注入的 `goroutine` 和 `sched` 包运行时 hook 实现，不依赖 Go runtime patch。每个 Pass 遵循 `Before → Apply → After` 三阶段生命周期。当前激活的 Pass（按注册顺序）：
- **GoroutinePass**：`go` 语句处插入 `goroutine.Enter(gid, parentGid)` / `defer goroutine.Exit(gid)`
- **ChRecPass**：channel send/close 操作处插入 `sched.InstChBF` / `sched.InstChAF` 钩子
- **SelectPass**：select 分支中的 send 操作插入 `sched.InstChSelectAF` 钩子
- **WgPass**：WaitGroup Add/Done 操作处插入 `sched.InstWgBF` / `sched.InstWgAF` 钩子
- **TestPass**：测试函数生成 `TestXxx_1` 包装，注入 `EnterMain` / `ExitMain` / `ParseInput` / `sched.ParseInput` / `PrintGoroutinePairs` 生命周期

> **注意**：`FunctionPass`（函数级 funcID 分配）在注册时已被注释掉；函数级 `callstack` 包已移除。

### 2. 静态 GID 机制

Goroutine ID（gid）在**编译期**通过 `文件路径 hash << 32 + 原子递增` 生成（`opid` 字段），硬编码在插桩后的源码中。

与运行时动态分配的 OS 线程级 ID（GortID）不同，静态 gid 保证跨执行的一致性，使得基于父子关系的推断规则（邻接父子、邻接兄弟、纯结构兄弟）有稳定的语义基础。

### 3. 两阶段 Fuzzing 架构

预执行与 Fuzzing 的切换由 `Config.GortPhase` 统一控制（0=预执行，1=Fuzzing）。`CorpusGort` 和 `CorpusOp` 各自持有 `*uint32` 指针指向该字段，通过 `atomic` 读写。

**两 Corpus 生命周期对齐**：

```
                预执行收集          阶段转换               取种子      反馈
CorpusGort:   AddPair()     →  TryEndPreExec()      →  Get()  →  ApplySignals()
CorpusOp:     Add()         →  TryEndPreExec(cg)    →  Get()  →  ApplySignals() + OnGortCovered()
```

`TryEndPreExec` 由 monitor 循环每轮并列调用，各自在内部管理日志输出，无返回值。

#### Phase 1 — 预执行阶段（Pre-execution）

- 反复运行测试二进制（`Input=""`），`SCHED_DEBUG` 未设置 → `debugSched=true` → `[FB]` 日志输出到 stderr
- **stderr**：`ParseGortPairs` → goroutine 对（`[COVERED]`/`[SUSPECT]`）→ `corpusGort.AddPair()` 分类入 CoveredConPairs 或 SusConPairs
- **stderr**：`ParseGortPairs` → OpInfo（`[FB]chan:`/`[FB]wg:`）→ `corpusOp.Add()` 去重索引
- 采用不动点停止策略：连续 3 轮种子总数不变 或 达到最大轮次（默认 30）
- 阶段转换：`CorpusGort.TryEndPreExec` 设置 `cfg.GortPhase=1` + `RefillTryPairs`
- 转换后：`CorpusOp.TryEndPreExec` 从 `cg.CoveredConPairs` 一次性生成全部 OP 对 → `SusConPairs` → `RefillTryPairs`
- 种子按置信度 × 10 − 超时次数 × 2 的优先级排序

#### Phase 2 — Fuzzing 阶段

- 从 SusConPairs 按优先级选取种子，通过环境变量 `Input` / `InputOp` 传递给运行时
- 运行时 `goroutine.pointControl` 实现 goroutine 间断点同步：双方 goroutine 在入口处 rendezvous，同时到达才放行
- `sched.InstChBF`/`InstWgBF` 实现操作级断点同步
- 从 **stdout** 解析 `{COVERED}` / `{TIMEOUT}`（goroutine 级）和 `{COVERED_OP}` / `{TIMEOUT_OP}`（OP 级）信号
- 超时 5 次后温和淘汰至 InfeasiblePairs
- 无覆盖时 selectNum 翻倍（上限 64）扩大搜索范围
- goroutine 对 COVERED → `corpusOp.OnGortCovered()` 增量生成对应 OP 对

### 4. 并发对推断规则

共 4 条规则，在 `PrintGoroutinePairs()`（`pkg/goroutine/infer.go`）中依次执行后统一去重合并。去重按 `(gid1,gid2)` key，冲突时保留高置信度。gid=0（主 goroutine）不参与输出。

| # | 规则 | 函数 | 置信度 | 输出标签 | 说明 |
|---|------|------|--------|----------|------|
| 直接观测 | 时间重叠检测 | `DetectGoroutineOverlaps` | 1.0 | `[COVERED]` | 计算每个 gid 的时间范围 `[minStart, maxEnd]`，O(n²) 两两比较重叠 |
| 纯结构兄弟 | 同父 children 全配对 | `InferAllSiblingPairs` | 0.5 | `[SUSPECT]` | 不依赖观测，childMap 中同一 parent 的 children（≥2）两两配对，来源 `inferred_sibling` |
| 邻接父子 | COVERED 对 → 父子一跳 | `InferAdjacentPairs` | 0.5/0.3 | `[SUSPECT]` | 基于直接观测结果向外扩展：向上（parent×对方）0.5，向下（children×对方）0.3，来源 `inferred_adjacent` |
| 邻接兄弟 | COVERED 对 → 兄弟一跳 | `InferSiblingAdjacentPairs` | 0.5 | `[SUSPECT]` | 基于直接观测结果向兄弟方向扩展（ga兄弟×gb, ga×gb兄弟），来源 `inferred_sibling` |

**邻接父子置信度差异**：向上（parent）0.5 > 向下（children）0.3，因为子节点启动时机不确定，与重叠窗口的关系弱于 parent 方向。

### 5. 信号反馈机制

**信号类型**（`pkg/feedback/signal.go`）：

| 信号 | Kind | stdout 格式 | 含义 |
|------|------|------------|------|
| `{COVERED}` | `SignalGortCovered` | `{COVERED} {gid1, gid2}` | goroutine 对调度成功 |
| `{TIMEOUT}` | `SignalGortTimeout` | `{TIMEOUT} {gid1, gid2}` | goroutine 对调度超时 |
| `{COVERED_OP}` | `SignalOpCovered` | `{COVERED_OP} {opId1, opId2}` | 操作对调度成功 |
| `{TIMEOUT_OP}` | `SignalOpTimeout` | `{TIMEOUT_OP} {opId1, opId2}` | 操作对调度超时 |

**ApplySignals 处理流程**（goroutine 和 OP 两个 Corpus 完全对称）：

1. 分类 coveredSigKeys / timeoutSigKeys
2. 超时计数：TryPairs 中匹配 timeoutSigKeys → 累计超时，≥5 次 → SusConPairs → InfeasiblePairs（淘汰）
3. 计算 `COVERED ∩ TryPairs` = intersection
4. 无覆盖但有超时 → selectNum 翻倍扩大搜索范围（上限 64）
5. intersection 中的对：从 SusConPairs **删除**，加入 CoveredConPairs（晋升）
6. RefillTryPairs() 从剩余 SusConPairs 按 score 重选种子

**goroutine 特有的反馈链路**：`ApplySignals` 返回本轮新覆盖的 goroutine 对列表 → `corpusOp.OnGortCovered()` 增量生成对应 OP 对种子。

## 构建与运行

### 构建

```shell
go build -o ./bin ./cmd/...
```

生成 `bin/fuzz.exe` 和 `bin/inst.exe` 两个二进制文件。

### 完整测试流程

```shell
# 1. 插桩：对被测项目源码注入调度代码
./bin/fuzz --task inst --path your_project_path

# 2. 编译：go test -race -c 生成测试二进制 → testbins/
./bin/fuzz --task bins --path your_project_path

# 3. Fuzzing 测试：运行检测
./bin/fuzz --task full --path testbins/
```

### 关键命令行参数

| 参数 | 说明 |
|------|------|
| `--task` | `inst` / `bins` / `full` / `lite` |
| `--path` | 目标路径 |
| `--timeout` | 单次执行超时（秒，默认 30） |
| `--recovertimeout` | 恢复超时（秒，默认 100） |
| `--max` | 最大并发 worker 数 |
| `--func` | 指定测试函数名 |
| `--feature` | `full`（完整）/ `fb`（无反馈）/ `mu`（无变异） |
| `--llevel` | 日志级别：`debug` / `info` / `normal` |
| `--check` | 泄漏检查位置：`inside` / `outside` |
| `--output` / `-o` | 输出目录 |

## 开发约定

### 日志系统

- `[FB]` 前缀：插桩注入的 `sched` 包通过 `print()` 内建函数输出的操作日志（`[FB]chan:` / `[FB]wg:`）
- 并发对 stderr 格式：`[COVERED]` / `[SUSPECT] gid1,gid2|file1:line1,file2:line2|confidence|sourceType`（由 `goroutine.PrintGoroutinePairs()` 输出）
- Fuzzing 信号 stdout 格式：`{COVERED} {gid1, gid2}` / `{TIMEOUT} {gid1, gid2}`（goroutine 级），`{COVERED_OP}` / `{TIMEOUT_OP}`（OP 级）

### 并发安全规范

- `CorpusGort` 和 `CorpusOp` 使用 `sync.Mutex`（非 RWMutex）保护内部 map 和计数器
- `sync.Map` 的复合操作（Load + Store）需要额外 mutex 保护
- barrierGate 使用 atomic arrived 计数器 + release channel 保证并发安全
- 项目对锁开销敏感，尽量减少临界区范围

### Pass 开发模式

新增插桩 Pass 的步骤：
1. 在 `pkg/inst/passes/` 创建新文件
2. 实现 `InstPass` 接口（Before / GetPreApply / GetPostApply / After）
3. 在 registry 中注册 pass 名称和构造函数
4. 在 `cmd/inst/inst.go` 的注册列表中按序添加

### 测试注意事项

- Windows 上测试二进制以 `.exe` 结尾，`exec.Command` 传入路径需包含 `.exe` 后缀
- Race Detector 的插桩可能与自定义调度产生交互，可能掩盖部分竞态
- `go test -race -c` 编译后的二进制通过 `-test.run` 参数指定测试函数
- 预执行阶段和 Fuzzing 阶段共用一个 Monitor 循环，通过 `cfg.GortPhase` 标志区分

## 已知限制

- OP 级别 Fuzzing 的 `[FB]` 操作日志通过 `print()` 内建函数输出到 stderr，仅在 `SCHED_DEBUG` 未设置时生效；fuzzing 执行阶段设置 `SCHED_DEBUG=1` 后会关闭日志输出
- select 语句的 AST 插桩存在结构限制，某些复杂 select 无法完整插桩；select 分支中的操作仅插入 AF 钩子，无法作为 OP 对的 Next 方（只能做 Pre）
- Bug 结果汇总模块（`pkg/bug`）尚未完成，panic/data race 仅在日志中打印
- Fuzzing 种子选择可能出现饥饿问题（RefillTryPairs 中 TODO 标注）
- `cfg.MaxWorker` 被硬编码为 4（monitor.go:113），用户配置不生效
- 大量 Config 字段未被使用（`InitTurnCnt`、`UseCoveredSched`、`UseStates`、`UseAnalysis`、`UseGuide`、`SingleCrash`）
- `Monitor.Start` 始终返回 `(false, []string{})`，bug 发现无返回路径

## 开发注意事项
- 实现功能时，多去复用现有的代码，开发风格要保持统一
- 完成功能时，给出相应的变动总结
