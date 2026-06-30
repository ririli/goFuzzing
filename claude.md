# GoPie - Golang 并发 Fuzzing 测试工具

## 项目概述

GoPie 是一个基于 Fuzzing 的 Golang 数据竞争检测工具。通过对被测代码进行 **双层插桩**（Runtime Patch + AST Instrumentation）
，在测试执行时收集并发行为信息，构建 goroutine 调用顺序图，并据此引导 Fuzzing 变异策略来系统性地检测数据竞争、panic，当前项目采用goroutine作为颗粒度，函数颗粒度代码虽然保留但未使用。

## 技术栈

- **语言**: Go 1.19.1
- **模块名**: `toolkit`
- **核心依赖**: `go-flags`（命令行解析）、`testify`（测试框架）、`goleak`（goroutine 泄漏检测）、`x/tools`（Go AST 工具集）、`go-echarts`（可视化）

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
│   │   ├── instctx.go            # 插桩上下文（解析、类型检查、gid 分配）
│   │   ├── registry.go           # Pass 注册中心（Register/GetNewPassInstance）
│   │   ├── run.go                # Pass 执行引擎（Before → Apply → After 生命周期）
│   │   ├── type.go               # InstPass / InstContext 类型定义
│   │   ├── err.go                # 错误类型
│   │   ├── util.go               # 工具函数
│   │   └── passes/               # 各类插桩 Pass 实现
│   │       ├── func.go           # 函数入口/出口 Trace 插桩
│   │       ├── gort.go           # goroutine Enter/Exit 插桩
│   │       ├── channel.go        # channel 操作日志插桩
│   │       ├── select.go         # select 语句日志插桩
│   │       ├── waitgroup.go      # WaitGroup 操作日志插桩
│   │       ├── global.go         # 全局变量相关插桩
│   │       ├── test.go           # 测试函数包装
│   │       └── utils.go          # Pass 公共工具函数
│   ├── fuzzer/                   # Fuzzing 引擎
│   │   ├── config.go             # Config 配置（DefaultConfig / GokerConfig）
│   │   ├── corpus.go             # CorpusPair：并发对种子管理核心
│   │   ├── corpus_gort.go        # CorpusGort：goroutine 级别种子管理
│   │   ├── corpus_op.go          # CorpusOp：操作级别种子管理（已禁用）
│   │   ├── executor.go           # Executor：测试二进制执行与输出流式处理
│   │   ├── monitor.go            # Monitor：主监控循环（预执行 + Fuzzing 两阶段）
│   │   ├── utils.go              # 工具函数
│   │   └── visitor.go            # 文件系统遍历
│   ├── goroutine/                # 运行时 Goroutine 生命周期追踪
│   │   ├── gort.go               # Enter/Exit/pointControl 核心调度逻辑
│   │   ├── env.go                # GoroutineTracker 实现
│   │   └── overlap.go            # 时间重叠检测（Rule 0/2/3 推断）
│   ├── callstack/                # 运行时函数调用栈追踪
│   │   ├── trace.go              # Trace/pointControl 核心逻辑
│   │   ├── functionNode.go       # 调用树节点（FunctionNode）
│   │   ├── timeoverlap.go        # 函数级时间重叠分析
│   │   ├── susconpair.go         # 可疑并发对推断与输出
│   │   ├── parseinput.go         # Input 环境变量解析
│   │   └── env.go                # CallStackCollector 实现
│   ├── feedback/                 # 反馈信号类型定义
│   │   ├── type.go               # SuspiciousPairInfo / CoverageSignal 等核心类型
│   │   ├── signal.go             # 信号解析（ParseGortPairs / ParseSignals）
│   │   ├── format.go             # 格式化和序列化
│   │   ├── type_gort.go          # GortPairInfo / InputPair 类型
│   │   └── type_op.go            # OpInfo 类型
│   ├── sched/                    # 运行时调度原语（[FB] 日志输出）
│   │   ├── sched.go              # Channel/WG/Mutex 操作的 [FB] 日志
│   │   └── env.go                # 环境变量读取
│   ├── trace/                    # 全局追踪日志
│   ├── seed/                     # 种子分析
│   ├── compiler/                 # 编译工具
│   ├── bug/bug.go                # Bug 结果汇总（待实现）
│   ├── yield/yield.go            # 主动让出机制
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

### 1. 双层插桩

GoPie 采用双层插桩架构，从两个层面捕获并发行为：

**第一层 — Runtime Patch（运行时层）**：
修改 Go runtime / sync / time 包源码，在 goroutine 创建、channel 收发、互斥锁、WaitGroup 等并发原语关键位置输出 `[FBSDK]` 日志，提供运行时级别的并发行为记录。

**第二层 — AST Instrumentation（用户代码层）**：
通过 AST Pass 系统对用户源码进行静态插桩。每个 Pass 遵循 `Before → Apply → After` 三阶段生命周期。插桩内容包括：
- 函数入口插入 `defer callstack.Trace(funcID)()`
- `go` 语句处插入 `goroutine.Enter(gid)` / `defer goroutine.Exit(gid)`
- channel 操作处插入 `sched` 包日志调用
- WaitGroup 操作处插入追踪日志
- select 语句各分支插入操作日志

### 2. 静态 GID 机制

Goroutine ID（gid）在**编译期**通过 `文件路径 hash << 32 + 原子递增` 生成，硬编码在插桩后的源码中。

与运行时动态分配的 OS 线程级 ID（GortID）不同，静态 gid 保证跨执行的一致性，使得 Rule 2（父子关系推断）和 Rule 3（自配对推断）有稳定的语义基础。

### 3. 两阶段 Fuzzing 架构

#### Phase 1 — 预执行阶段（Pre-execution）
- 反复运行测试二进制，从 **stderr** 解析并发种子信息
- 种子来源：直接观测的时间重叠对、父子关系推断、自配对推断、共享对象访问推测
- 采用不动点停止策略：连续 3 轮种子总数不变 或 达到最大轮次（默认 30）
- 种子按置信度 × 10 − 超时次数 × 2 的优先级排序

#### Phase 2 — Fuzzing 阶段
- 从 SusConPairs 按优先级选取种子，通过环境变量 `Input` 传递给运行时
- 运行时 pointControl 实现 goroutine 间断点同步：后执行的 goroutine 等待先执行的 goroutine 完成
- 从 **stdout** 解析 `{COVERED}` / `{TIMEOUT}` 信号进行反馈
- 交替反转策略：每两次执行交换 goroutine 先后顺序
- 超时 5 次后温和淘汰至 InfeasiblePairs
- 无覆盖时 selectNum 翻倍（上限 64）扩大搜索范围

### 4. 并发对推断规则

| 规则 | 来源 | 置信度 | 输出标签 |
|------|------|--------|----------|
| Rule 0 | 执行时间重叠 | 直接观测 | `[COVERED]` |
| Rule 1 | 共享对象（同 channel/WG） | 0.8 | `[SUSPECT]` |
| Rule 2 | 父子 goroutine 关系 | - | `[SUSPECT]` |
| Rule 3 | 同 gid 多实例自配对 | - | `[SUSPECT]` |

### 5. 信号反馈机制

- `COVERED` 信号：种子从 SusConPairs → CoveredConPairs，selectNum 不变
- `TIMEOUT` 信号：累计超时计数，≥5 次 → InfeasiblePairs（温和淘汰）
- 无覆盖 + 有超时 → selectNum 翻倍（不立即淘汰，给种子更多机会）

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

- `[FBSDK]` 前缀：Runtime Patch 层输出
- `[FB]` 前缀：AST Instrumentation 层输出
- Fuzzing 信号 stdout 格式：`{COVERED} {preID, nextID}` / `{TIMEOUT} {preID, nextID}`
- stderr 并发对格式：`[COVERED]` / `[SUSPECT] gid1,gid2|file1:line1,file2:line2|confidence|sourceType`

### 并发安全规范

- `CorpusPair` 及其子类使用 `sync.RWMutex` 保护内部 map
- `sync.Map` 的复合操作（Load + Store）需要额外 mutex 保护
- pointControl 中的 waiter channel 使用 `sync.Map` + 原子操作保证安全
- 项目对锁开销敏感，尽量减少临界区范围

### Pass 开发模式

新增插桩 Pass 的步骤：
1. 在 `pkg/inst/passes/` 创建新文件
2. 实现 `InstPass` 接口（Before / GetPreApply / GetPostApply / After）
3. 在 registry 中注册 pass 名称和构造函数
4. 在 `run.go` 的 pass 列表中按序添加

### 测试注意事项

- Windows 上测试二进制以 `.exe` 结尾，`exec.Command` 传入路径需包含 `.exe` 后缀
- Race Detector 的插桩可能与自定义调度产生交互，可能掩盖部分竞态
- `go test -race -c` 编译后的二进制通过 `-test.run` 参数指定测试函数
- 预执行阶段和 Fuzzing 阶段共用一个 Monitor 循环，通过 `corpusGort.done` 标志区分

## 已知限制

- OP 级别调度已禁用，当前仅使用 goroutine 级别调度
- select 语句的 AST 插桩存在结构限制，某些复杂 select 无法完整插桩
- Bug 结果汇总模块（`pkg/bug`）尚未完成
- Fuzzing 种子选择可能出现饥饿问题（RefillTryPairs 中 TODO 标注）
