# GoPie — Go 并发模糊测试工具

## 项目概述

GoPie 是一个 Go 语言的并发测试/模糊测试工具，通过 AST 插桩在运行时注入断点控制与调用追踪，并利用反馈驱动的模糊测试策略，系统性探索并发调度空间，检测数据竞争、channel/WaitGroup 误用等并发缺陷。

## 技术栈

- **Go 1.19** — 开发语言

## 模块名

Go module 名为 `toolkit`（非 `gopie`）。所有内部包的导入路径均以 `toolkit/` 开头。

## 目录结构

```
.
├── cmd/
│   ├── fuzz/          # 主 CLI 工具入口（inst/bins/full/lite 四个任务）
│   │   ├── main.go    # 命令行解析与任务路由
│   │   ├── inst.go    # 插桩任务：并发调用 inst 工具处理 .go 文件
│   │   ├── bins.go    # 编译任务：用 go test -race -c 生成测试二进制
│   │   ├── full.go    # 完整模糊测试：扫描所有二进制+测试函数并发执行
│   │   └── lite.go    # 轻量模式：单二进制+单函数模糊测试
│   ├── inst/          # 独立插桩工具（被 fuzz inst 任务调用）
│   │   └── inst.go    # 注册 5 个 Pass 并处理单文件/目录
│   ├── flags.go       # 插桩工具的 CLI 参数
│   └── utils.go       # 共享工具（ListFiles, ListTests, HandleSrcFile）
├── pkg/
│   ├── inst/                  # 插桩框架核心
│   │   ├── instctx.go         # InstContext：AST 解析+类型检查的上下文
│   │   ├── type.go            # InstPass 接口定义 + PassRegistry
│   │   ├── registry.go        # Pass 注册管理
│   │   ├── run.go             # Run(): 按顺序执行 Pass，调用 astutil.Apply
│   │   ├── util.go            # AddImport / DumpAstFile 工具
│   │   └── passes/            # 具体插桩 Pass 实现
│   │       ├── func.go        # FunctionPass: 在每个函数体前插入 defer Trace(id)()
│   │       ├── channel.go     # ChRecPass: channel send/close 的 BF/AF 插桩
│   │       ├── select.go      # SelectPass: select 中 channel 操作的 AF 插桩
│   │       ├── waitgroup.go   # WgPass: sync.WaitGroup Add/Done 插桩
│   │       ├── test.go        # TestPass: 生成 _1 包装测试函数，注入 ParseInput 等
│   │       ├── utils.go       # AST 节点生成工具函数
│   │       └── global.go      # 全局 id_map（pos → funcID 映射）
│   ├── callstack/             # 运行时门面包（被测代码唯一入口）
│   │   └── trace.go           # 组装 calltree + overlap + breakpoint 三大子系统
│   ├── calltree/              # 调用树收集
│   │   ├── collector.go       # CallStackCollector: 按 goroutine 记录函数 enter/exit
│   │   └── node.go            # FunctionCallNode: 携带时间戳的调用节点
│   ├── breakpoint/            # 断点/等待控制（两种策略）
│   │   ├── strategy.go        # Strategy 接口（PointControl / ParseInput / ParseSusPairs）
│   │   ├── config.go          # Config: 单栏策略（A 等 B）
│   │   ├── barrier.go         # BarrierConfig: 双栏策略（A 和 B 同时到达才放行）
│   │   ├── control.go         # PointControl: 单栏等待逻辑实现
│   │   └── waiter.go          # Waiter channel 池
│   ├── overlap/               # 时间重叠检测
│   │   ├── detect.go          # DetectFunctionOverlaps: 跨 goroutine 函数时间重叠分析
│   │   ├── infer.go           # InferSuspiciousPairs: 邻接推断规则（父/子节点）
│   │   └── types.go           # ConPairFunc / TimeOverlap 类型
│   ├── fuzzer/                # 模糊测试引擎
│   │   ├── config.go          # Config: 所有模糊测试可调参数
│   │   ├── monitor.go         # Monitor.Start: 主循环，worker 池 + 信号收集
│   │   ├── corpus.go          # CorpusPair: 并发对种子管理（预执行+自适应选取）
│   │   ├── corpus_op.go       # CorpusOp: 操作对种子管理
│   │   ├── executor.go        # Executor.Run: 执行测试二进制，注入环境变量
│   │   ├── visitor.go         # Visitor（预留扩展）
│   │   └── utils.go           # Hash32
│   ├── feedback/              # 反馈信号解析
│   │   ├── signal.go          # CoverageSignal / SignalKind
│   │   ├── type.go            # SuspiciousPairInfo / InputPair
│   │   ├── type_op.go         # OpInfo / OpPair / OpKind / DangerType
│   │   └── format.go          # ParseStdPairs / ParseSignals
│   ├── sched/                 # 运行时调度拦截
│   │   ├── sched.go           # InstChBF / InstChAF / InstWgBF / InstWgAF
│   │   └── env.go             # Config: preOpMap / waitMap
│   ├── bug/                   # 缺陷收集
│   ├── yield/                 # 协程让出
│   └── utils/
│       ├── gofmt/             # 语法检查（HasSyntaxError）
│       └── hash/              # 哈希工具
├── testdata/                  # 测试数据（被测项目源码）
│   ├── gobench/nonblocking/   # 各开源项目的非阻塞 bug 测试用例
│   └── myTest/                # 自定义测试用例
├── testbins/                  # 编译产物目录（.exe 测试二进制）
├── script/                    # 构建/部署脚本
└── zresult/                   # 测试结果输出
```

## 核心架构流程

### 1. 插桩阶段（`inst` 任务）

```
源码 → InstContext(解析AST + 类型检查)
     → 依次执行 5 个 Pass（astutil.Apply 遍历 AST）:
       FunctionPass → ChRecPass → SelectPass → WgPass → TestPass
     → DumpAstFile (格式化输出 + 语法检查回退)
```

插桩注入的内容：
- **每个函数体开头**: `defer callstack.Trace(funcID)()` — 用于断点控制 + 调用树记录
- **channel send/close 前后**: `sched.InstChBF/InstChAF(opId, ch, funcId, opType)` — 用于操作对调度
- **select 中的 channel 操作**: `sched.InstChSelectAF(opId, ch, funcId, opType)` — 仅 AF，无 BF
- **WaitGroup Add/Done 前后**: `sched.InstWgBF/InstWgAF(opId, wg, funcId, opType)`
- **测试函数**: 生成 `TestXxx_1` 包装函数，注入 `callstack.ParseInput()` + `sched.ParseInput()` + `defer callstack.PrintSusConPairs()`

### 2. 编译阶段（`bins` 任务）

对每个包含 .go 文件的目录执行 `go test -race -o <output> -c .`，生成带竞态检测的测试二进制文件到 `testbins/` 目录。

### 3. 模糊测试阶段（`full` 任务）

```
扫描 testbins/ → 列出每个二进制的测试函数 → 并发执行 Monitor.Start
  └── Monitor 工作循环:
      ├── CorpusPair.Get() 获取本轮种子（函数对）
      ├── CorpusOp.Get() 获取操作对种子
      ├── Executor.Run() 执行测试二进制（通过环境变量 Input/InputOp 注入种子）
      ├── 收集 stderr: 调用树/重叠信息 → corpusPair.AddPair / corpusOp.Add
      ├── 收集 stdout: 调度信号 {COVERED}/{TIMEOUT} → ApplySignals
      ├── 预执行阶段: TryEndPreExec 判断种子是否收敛
      └── Fuzzing 阶段: 自适应调整 selectNum，反馈驱动种子替换
```

### 关键设计

- **双栏 vs 单栏**: 通过环境变量 `BARRIER_MODE=double` 切换。单栏模式 A→B（有向），双栏模式 A↔B（双向等待，需同时到达）
- **邻接推断**: 观测到 (a,b) 并发 → 推断 (a.parent, b), (a, b.parent), (a.children, b), (a, b.children) 也可能并发
- **自适应种子选取**: selectNum 随超时自动翻倍（上限 64），超时 ≥5 次移入 InfeasiblePairs
- **种子反转**: 每 2 次执行反转一次 pair 方向（A,B → B,A），探索双向调度

## 构建与运行

```bash
# 编译
go build -o ./bin ./cmd/...

# 插桩
./bin/fuzz --task inst --path <project_path>

# 编译测试二进制
./bin/fuzz --task bins --path <project_path> -o <output_dir>

# 运行模糊测试
./bin/fuzz --task full --path <testbins_path> --llevel debug > log.txt
```

## 环境变量

被测测试进程运行时通过环境变量接收配置：
- `Input`: 可疑函数对 `(id1,id2)(id3,id4)...`
- `InputOp`: 操作对 `(opId1,opId2)...`
- `TIMEOUT`: 执行超时（秒）
- `RECOVER_TIMEOUT`: 恢复超时
- `RECORD_STACK`: 设为 1 时跳过调用栈记录，仅断点控制
- `BARRIER_MODE`: `double` 启用双栏策略
- `SCHED_DEBUG`: 控制操作详情日志

## 编程约定

- 所有 Pass 实现 `InstPass` 接口（`Before` / `GetPreApply` / `GetPostApply` / `After`）
- 插桩代码通过 AST `InsertBefore`/`InsertAfter`/`Replace` 注入，需 recover panic
- 插桩后检查语法错误，失败则回退到原始内容
- `callstack` 包是被测代码唯一入口，它内部组装三个子系统
- fuzzer 通过 channel 通信（`resCh` 结果 / `logCh` 日志 / `cancel` 退出信号）
- 包内 `init()` 函数负责读取环境变量初始化配置
