# GoPie 项目指南

## 项目定位

GoPie 是一个面向 Go 并发缺陷实验的研究原型。它并不修改 Go runtime，也不是通用的输入数据 Fuzzer。当前主链路是：

1. 使用 Go AST Pass 原地改写目标源码，注入 goroutine 生命周期和 channel/WaitGroup 操作 hook。
2. 使用 `go test -race -c` 编译测试二进制，后续只枚举并运行插桩测试 `TestXxx_1`。
3. 预执行测试，收集 goroutine 生命周期记录、父子关系和部分同步对象操作；直接重叠观测只使用已结束实例。
4. 从直接观测和结构推断得到 goroutine 对，再派生可能触发 panic 的操作对。
5. 通过环境变量下发调度对，搜索新的 goroutine/操作执行关系。
6. 复用 Go race detector 的 stderr 和 panic 文本作为缺陷线索。

这里的 “fuzzing” 主要是并发调度搜索，不会变异普通函数参数或测试输入。代码也没有构建完整的调用顺序图；运行时保存的是静态 goroutine 创建点对应的动态实例、生命周期区间及父子列表。

> 当前 Monitor 只记录 race/panic 日志，所有退出路径仍返回 `(false, []string{})`。因此命令最终打印的 `PASS` 不能证明未发现缺陷，必须检查完整日志。

## AI 交互与修改原则

- 以代码实际行为为准；README、注释、历史结果和字段名都可能落后于实现。
- 对错误假设应直接指出并给出代码依据，不要为了迎合需求而确认不实结论。
- 修改应保持范围小并复用现有结构；不要顺手改写基准、二进制或历史结果。
- 插桩会原地覆盖源码且不幂等。除非任务明确要求，不要对 `testdata/gobench/nonblocking_origin` 或用户未备份的源码执行插桩。
- 不要把 `bin/`、`testbins/`、`raceResult/`、`zgortResult/` 当作可靠的跨平台构建产物或测试 oracle。
- 完成代码变更后给出变更总结和实际验证结果；未运行或未通过的验证必须明确说明。

## 技术基线

- 模块名：`toolkit`
- `go.mod` 语言版本：Go 1.19；README 说明项目最初使用 Go 1.19.1 实现，但 patch 版本不是模块约束。
- 直接依赖：
  - `github.com/jessevdk/go-flags v1.5.0`：CLI flags。
  - `golang.org/x/tools v0.2.0`：`astutil.Apply` 等 AST 工具。
- 间接依赖：`golang.org/x/sys v0.1.0`。
- 测试只使用标准库 `testing`，没有直接依赖 `testify`。

## 目录与职责

```text
gopie/
├── cmd/
│   ├── fuzz/
│   │   ├── main.go        # --task 路由：inst / bins / full / lite
│   │   ├── inst.go        # 并发调用独立 inst 二进制
│   │   ├── bins.go        # 按目录 go test -race -c
│   │   ├── full.go        # 扫描二进制并为每个 Test*_1 启动 Monitor
│   │   └── lite.go        # 单二进制/单测试入口
│   ├── inst/inst.go       # 独立 AST 插桩器
│   ├── flags.go           # 独立 inst flags
│   └── utils.go           # 文件/测试枚举、AST 写回
├── pkg/
│   ├── inst/              # InstContext、Pass registry、执行和写回
│   │   └── passes/        # gort/channel/select/waitgroup/test Pass
│   ├── goroutine/         # 生命周期追踪、双栏调度、并发对推断
│   ├── sched/             # 操作日志、pre -> next 操作调度
│   ├── feedback/          # stderr/stdout 协议解析和数据类型
│   ├── fuzzer/            # Config、Executor、Monitor、两个 Corpus
│   ├── bug/               # 仅有 TODO，尚无结果汇总实现
│   └── utils/             # gofmt/hash 辅助
├── testdata/
│   ├── gobench/nonblocking_origin/ # 35 个未插桩对照案例
│   ├── gobench/nonblocking/        # 对应的 35 个已插桩案例
│   └── myTest/                     # 小型实验样例，含历史残留
├── bin/                    # 当前提交的是 Windows x86-64 PE 工具
├── testbins/               # 当前提交的是 Windows x86-64 PE 测试二进制
├── script/run_all.ps1      # 7 组 GoBench 的 bins + full PowerShell 脚本
├── raceResult/             # 历史汇总，当前代码没有自动生成入口
└── zgortResult/            # 历史 full 输出
```

GoBench 子集包含 cockroach 2、etcd 4、grpc 4、istio 4、kubernetes 12、moby 3、serving 6，共 35 个案例。`testdata/gobench/README.md` 是不完整的上游残留说明，其中提到的 Makefile、artifact.pdf 等并不存在。

## 构建与验证

### 构建命令

```shell
go build -o ./bin ./cmd/...
```

Unix 产物通常是 `bin/fuzz` 和 `bin/inst`，Windows 才是 `fuzz.exe` 和 `inst.exe`。仓库当前提交的 `bin/*.exe` 与 `testbins/**/*.exe` 都是 Windows x86-64 PE；在 macOS/Linux 上必须本机重建。

### 当前可靠的核心测试

```shell
go test ./cmd ./pkg/...
```

该命令当前通过。不要把 `go test ./...` 作为现状下的绿色基线：

- `cmd/fuzz/inst.go` 当前会触发 vet：`fmt.Println arg list ends with redundant newline`。
- `cmd/fuzz/main_test.go` 使用硬编码 Windows 绝对路径；找不到测试时 `Lite` 可能无限空转。
- Go 会跳过名为 `testdata` 的目录，因此 `./...` 本来也不会验证 GoBench fixtures。
- `pkg/goroutine`、`pkg/sched`、`pkg/inst/passes` 和 `pkg/utils/hash` 当前没有单元测试。
- `testdata/testinst`、`testdata/myTest/recycle` 仍引用已删除的 `toolkit/pkg/callstack`，不代表当前架构。

需要只检查 `cmd/fuzz` 是否可编译时，可使用：

```shell
go test -vet=off ./cmd/fuzz -run '^$'
```

## CLI 实际行为

### 推荐实验流程

确保目标源码处于可恢复状态，然后执行：

```shell
# 1. 原地插桩目标源码
./bin/fuzz --task inst --path path/to/target

# 2. 将每个含 Go 文件的目录编译成 -race 测试二进制
./bin/fuzz --task bins --path path/to/target -o testbins/local

# 3. 对仅含本机测试二进制的目录运行调度搜索
./bin/fuzz --task full --path testbins/local
```

插桩固定加入 `toolkit/pkg/goroutine` 和 `toolkit/pkg/sched`。仓库内 fixtures 可直接解析；外部模块必须自行让这两个 import 可见，例如在目标 `go.mod` 中增加对 `toolkit` 的 `require` 和指向本仓库的本地 `replace`。工具不会修改目标 `go.mod`。

### `fuzz` flags

| 参数 | 实际作用域和行为 |
|---|---|
| `--task` | 必填路由：`inst` / `bins` / `full` / `lite`。 |
| `--path` | `inst/bins` 时是源码树；`full` 时是测试二进制目录；`lite` 时是单个测试二进制。 |
| `--output` / `-o` | 仅 `bins` 使用，默认 `testbins`。 |
| `--func` | 仅 `lite` 使用，指定 `TestXxx_1`。`full` 忽略。 |
| `--timeout` | 仅 `lite` 解析为 Executor 子进程秒数；缺省值会以 0 覆盖 Config，导致 context 立即过期。`full` 忽略并固定为 30 秒。 |
| `--recovertimeout` | 仅 `lite` 解析；`full` 固定为 200 秒。值只被写入无人读取的 `RECOVER_TIMEOUT` 环境变量。 |
| `--max` | `full` 中控制并行 Monitor 数，默认 24；每个 Monitor 内仍固定 4 worker。`lite --func` 中控制重复启动的 Monitor 数，必须大于 0。 |
| `--llevel` | `debug` / `info` / 其他值（normal 行为），供 `full/lite` 使用。 |
| `--feature` | 仅 `full`。`mu` 令 `UseMutate=false`，只停止应用反馈；`fb` 设置了未被读取的 `UseFeedBack=false`，当前无实际效果。 |
| `--check` | 传给 `TestPass.Pos`，但该字段未被读取，当前无实际效果。 |

`full` 对 `--path` 下所有 Walk 结果调用 `ListTests`，包括根目录、子目录和杂项文件；路径最好是只含本机测试二进制的平坦目录。`ListTests` 通过 `-test.list _1` 只收集名称以 `Test` 开头、以 `_1` 结尾的测试。

`lite` 在未提供数值参数时会用 0 覆盖 `DefaultConfig`。尤其是带 `--func` 却没有正数 `--max` 时不会启动任务且会无限空转，不要依赖 Config 中看似存在的默认值。

### 独立 `inst` flags

`cmd/inst` 支持 `--file`、`--out`、`--dir`、`--onlygoleak`、`--checkpos`：

- `--dir` 优先于 `--file`，缺省原地覆盖。
- 目录模式没有禁止 `--out`，多个源文件会写向同一个目标，不能这样组合使用。
- `--onlygoleak` 未使用；`--checkpos` 虽传入 `TestPass.Pos`，也未使用。

## AST 插桩

### Pass 框架

`InstContext` 为每个源文件保存原始内容、独立 `token.FileSet`、AST、单文件 `types.Info`、metadata 和 ID 计数器。类型检查错误只记录日志，不会阻止插桩。

Registry 按注册顺序创建新的 Pass 实例；没有依赖解析。每个 Pass 都执行：

```text
Before -> astutil.Apply(pre, post) -> After
```

当前顺序固定为：

```text
gort -> channel -> select -> waitgroup -> test
```

`FunctionPass` 未注册；`passes/global.go` 的 `id_map` 目前只有写入，没有 `Find` 调用，属于遗留结构。

### 各 Pass 的真实覆盖范围

| Pass | 注入行为 | 当前边界/语义风险 |
|---|---|---|
| `GoroutinePass` | 把 `go f(args)` 包成匿名 goroutine，子 goroutine 调用 `Enter(gid,parent)` 并 defer `Exit(gid)`。 | 只有 `CurrentGid()` 实参在父 goroutine 求值；原函数值和参数被移入子 goroutine 求值，可能改变原 Go 语义。 |
| `ChRecPass` | 普通 send 前后插 `InstChBF/AF`；普通或 defer `close` 前后插 hook。 | 不处理 receive/make；`close` 参数必须是简单标识符；AF 会再次求值 channel 表达式。select send 的 BF 插入失败后由 SelectPass 处理。 |
| `SelectPass` | 只在 select 的 send case body 开头插 `InstChSelectAF`。 | 不处理 receive；只有 AF，所以该操作只能作为 OP 调度的 pre，不能作为 next。 |
| `WgPass` | 对表达式语句/defer 形式的 `Add/Done` 插 `InstWgBF/AF`。 | 不处理 `Wait`；未知 receiver 类型默认接受，可能误插自定义 `Add/Done`；`&receiver` 可能重求值或记录错误地址。 |
| `TestPass` | 浅复制原测试函数体，追加 `TestXxx_1` 并注入 main 生命周期、输入解析和结果打印 hook。 | 只识别参数名恰为 `t`、语法恰为 `*testing.T` 且名称不以 `_1` 结尾的函数；不是调用原测试。 |

defer close/WaitGroup 会被改成 defer 闭包，receiver/参数由“注册 defer 时求值”变成“执行 defer 时求值”，这同样可能改变被测程序语义。

### 测试包装函数

生成的 `TestXxx_1` 逻辑顺序为：

```go
goroutine.EnterMain()
defer goroutine.ExitMain()
goroutine.ParseInput()
sched.ParseInput()
defer goroutine.PrintGoroutinePairs()
// 浅复制的原测试体
```

defer 按 LIFO 执行，所以 `PrintGoroutinePairs` 在 `ExitMain` 之前运行。

### 静态 ID

goroutine 创建点 ID 和操作 ID 共用同一个“每文件”计数器：

```go
context.opid = uint64(hash.Hash32(sourcePath)) << 32
id := atomic.AddUint64(&context.opid, 1) // 第一个 ID 是文件前缀 + 1
```

这些 ID 不是动态 goroutine 实例 ID，也不是真正全局唯一：

- ID 依赖传给插桩器的原始路径字符串；相对/绝对路径、路径分隔符和 Pass/源码变化都会改变它。
- `Hash32` 使用 MD5 前 4 字节的小端 uint32 作为文件前缀；它修复了原先只使用首字节的问题，但 32 bit hash 仍不保证绝对无碰撞。
- 低 32 位计数器只在单个 `InstContext` 内递增，所有启用 Pass 共用。
- 插桩没有把创建位置传给 runtime；`CallLoc` 当前通常输出为 `:0`，不能依靠 ID 反查源码位置。
- 哈希修复只影响之后重新插桩生成的 ID；仓库中已经插桩的 fixtures 不会自动改号。

### 写回与幂等性

- `fuzz --task inst` 原地覆盖所有 `.go` 文件，不排除 vendor、generated、origin 或已插桩文件。
- 再次插桩会重复包装 go、重复插 hook，并从原 `TestXxx` 再生成同名 `TestXxx_1`；它不是幂等操作。
- 写回后只用 `go fmt` 检查语法，不能发现类型错误或重复声明。
- 独立 `inst` 多处忽略 `HandleSrcFile` 错误，`fuzz inst` 的成功统计可能误报。
- `DumpAstFile` 打开已有文件时没有 `O_TRUNC`，若新内容更短可能残留旧尾部。
- `fuzz inst` 在消费结果前先投递全部任务，结果 channel 容量 100；文件数达到约 117 时可能因 16 个 worker 同时阻塞而死锁。

## Goroutine 运行时与推断

### 生命周期

包初始化时创建全局 Config/Tracker，并且只读取一次 `RECORD_STACK`。`CurrentGid` 通过 `runtime.Stack` 解析的是 Go runtime goroutine ID，不是 OS 线程 ID，再由 `gidMap` 映射到静态创建点 ID。

`Enter` 的顺序是：

1. 先执行 `pointControl(gid)`。
2. 若未跳过记录，再写 `runtime goroutine ID -> static gid` 映射。
3. 记录 start 和 parent。

因此 barrier 等待时间不计入生命周期重叠窗口。`Exit` 删除映射，并结束该静态 gid 最近一个尚未结束的动态实例。

### 预执行并发对

`PrintGoroutinePairs` 固定 sleep 500ms 等待子 goroutine。直接重叠检测只使用已结束实例：它先把同一静态 gid 的所有完成实例压缩成 `[min(start), max(end)]`，再对不同 gid 做 O(n^2) 区间比较。这不是逐实例的精确重叠，实例间空档也会被算入，可能产生假阳性；500ms 后仍运行的实例不会进入直接重叠检测。不过结构推断读取的是 `childMap`，运行中实例已经登记的父子关系仍可能参与 sibling/adjacent 推断。

推断规则如下：

| 来源 | 函数 | 置信度 | 含义 |
|---|---|---:|---|
| 直接观测 | `DetectGoroutineOverlaps` | 1.0 | 聚合生命周期区间重叠，输出 `[COVERED]`。 |
| 纯结构兄弟 | `InferAllSiblingPairs` | 0.5 | 同一 parent 的 children 两两组合。 |
| 邻接父子 | `InferAdjacentPairs` | parent 方向 0.5，child 方向 0.3 | 从 observed 对向父/子一跳扩展。 |
| 邻接兄弟 | `InferSiblingAdjacentPairs` | 0.5 | 从 observed 对向双方兄弟扩展。 |

最终按无序 `(gid1,gid2)` 合并，保留更高置信度并过滤 gid 0。输出 map 未排序，顺序不稳定。

已知推断限制：

- `childMap` 按动态实例追加静态 child gid，不去重。同一 go 点执行多次时可能推断出 `(gid,gid)` 自配对。
- parent 取某静态 gid 第一个实例的 `ParentGid`；同一创建点在不同父上下文执行时可能不准确。
- `CallLoc` 没有被当前 `Enter` 路径填充，格式虽含 `file:line`，实际通常为空路径和 0 行。

### 双栏调度

`Input` 格式为 `(gid1,gid2)(gid3,gid4)...`。每对创建一个双方共享的 `barrierGate`，任一参与 gid 到达 `Enter` 时原子增加 arrived：

- 第二次总到达仅在 gate 尚未过期时关闭 release，并向 stdout 打印 `{COVERED}`；若第一次已超时，后续到达不会关闭 release 或补发 covered。
- 第一次到达最多等待 10ms，超时向 stdout 打印 `{TIMEOUT}`。

计数只看总到达次数，不验证是两个不同 ID。相同静态 go 点的两个动态实例可能替代配对另一方；gate 也是一次性状态。一次 Input 含多对时，同一 gid 会按 slice 顺序经过它参与的所有 gate。

## 操作级运行时

预执行时，AF hook 使用内建 `print` 向 stderr 输出：

```text
[FB]chan: obj=<addr>; opId=<id>; gid=<gid>; op=send|close;
[FB]chan: obj=<addr>; opId=<id>; gid=<gid>; op=send; select=1;
[FB]wg: obj=<addr>; opId=<id>; gid=<gid>; op=add|done;
```

对象地址来自运行时指针，只能在同一次进程执行的快照内比较。`CorpusOp` 在 phase 保持 0 时只接受第一批非空 OP 日志，后续 phase 0 结果会被跳过。

`MatchOpPair` 只生成同种类、同对象地址的三类有向危险组合：

- `close -> send`
- 不同 OpId 的 `close -> close`
- `done -> add`

OP 调度使用 `InputOp=(preId,nextId)...`：

- pre 的 AF 完成后把 `preId` 写入全局 `event`。
- next 的 BF 忙等 `LoadAndDelete(preId)`。
- 成功消费后打印 `{COVERED_OP}`；4 秒内未消费打印 `{TIMEOUT_OP}`。

事件是一次性消费。同一个 pre 被多个 next 依赖时，通常只有一个 next 能成功。`{COVERED_OP}` 只表示 next 的 BF 看到了 pre 已完成，不表示 next 操作已经完成，更不表示已经触发 bug。

## 两阶段 Corpus 与反馈

`Config.GortPhase` 是两个 Corpus 共享的原子阶段标志：

### Phase 0：预执行

- `CorpusGort.Get` 和 `CorpusOp.Get` 返回 nil，Executor 传递空 `Input/InputOp`。
- Executor 自身不追加 `RECORD_STACK` 和 `SCHED_DEBUG`，所以通常会收集生命周期、推断对和 `[FB]` 日志；但子进程继承父进程环境，启动工具前必须确保这两个变量未设为 1。
- 每次预执行还输出逐行 `[GORT_EDGE] {"parent":...,"child":...,"count":...}`。父进程按无序执行批次合并拓扑并集，允许一个 child 对应多个 parent；本轮未出现的旧边不会被删除。
- stderr 的 observed goroutine 对进入 `CorpusGort.CoveredConPairs`，inferred 对进入 `SusConPairs`。
- pair 与唯一 topology edge 总数连续 3 个已处理结果不变，或达到 `MaxPreExecRound`（默认 30），`CorpusGort` 将 phase 切到 1。边的动态次数变化不会单独延长预执行。
- 紧接着，`CorpusOp` 仅从当前 covered goroutine 对和第一批 OP 快照生成初始操作对。

这里没有 worker barrier。4 个并发预执行 worker 中已在途的结果可能在 phase 切换后返回，阶段边界不是严格批次边界。此时 `CorpusOp.Add` 的 phase 0 跳过条件已经失效，可能把其他进程的对象地址混入索引；初始 OP 全量生成又不会为已有 covered goroutine 对重新执行。

### Phase 1：调度搜索

- `CorpusGort.Get` 和 `CorpusOp.Get` 返回当前 `TryPairs`，序列化到 `Input/InputOp`。
- 只要 gort input 对象非 nil，Executor 就设置 `RECORD_STACK=1` 和 `SCHED_DEBUG=1`。
- 变量名具有误导性：`RECORD_STACK=1` 实际令 `skipRecord=true`，关闭生命周期、gidMap 和并发对打印；`Enter` 中位于判断前的 `pointControl` 仍会执行。
- `SCHED_DEBUG=1` 关闭 `[FB]` 详情日志；OP BF/AF 调度仍工作。
- stdout 信号只有在 `UseMutate=true` 时才应用到 corpus。
- 新覆盖的 goroutine suspect 会标记为 `fuzz_verified` 并晋升到 covered。
- `CorpusGort` 使用预执行缓存的多 parent 拓扑，从新 covered 对向 parent、child、sibling 各扩展一跳，产生 `fuzz_inferred_adjacent` / `fuzz_inferred_sibling` 新种子；整个过程只发生在 fuzz 父进程内。
- 新 covered 同时触发 `CorpusOp.OnGortCovered` 增量派生操作对。
- phase 切换后晚到的预执行边仍会合并，并基于全部 covered 对补做推断和刷新 `TryPairs`。

当前没有实现带采集的 discovery replay。fuzz 子进程仍不输出拓扑，因此增量推断只能使用预执行阶段已经观察到的拓扑边，不能发现初始预执行从未执行到的新 goroutine 路径。

### 选种与淘汰

- Goroutine 分数：`confidence * 10 - timeoutCount * 2`。
- OP 分数：固定基准 `5 - timeoutCount * 2`，不继承 goroutine 置信度。
- 两类 corpus 初始各选 1 个 suspect；无 covered 但有 timeout 时 `selectNum` 翻倍，上限 64。
- 单对累计 5 次 timeout 后移入 `InfeasiblePairs`。
- covered 信号只晋升当前 `TryPairs` 中匹配的项。
- 没有任何信号的高分种子不会降权，可能长期占据 `TryPairs`。
- Goroutine 信号按无序 gid 对匹配；OP 信号按 `pre -> next` 有向匹配。

两个 Corpus 都用 `sync.RWMutex`，但它们并不“完全对称”：来源、key 方向、评分、生成时机和返回值都不同。

## 输出协议

必须区分 stderr 的“预执行观测/推断”和 stdout 的“强制调度反馈”：

| 流 | 格式 | 含义 |
|---|---|---|
| stderr | `[COVERED] gid1,gid2\|loc1,loc2\|1.00\|observed;` | 生命周期聚合区间被直接观测为重叠。 |
| stderr | `[SUSPECT] gid1,gid2\|loc1,loc2\|confidence\|source;` | 结构推断的 goroutine 对。 |
| stderr | `[GORT_EDGE] {"parent":p,"child":c,"count":n}` | 单次预执行内聚合、排序后的静态父子边和动态出现次数。 |
| stderr | `[FB]chan: ...` / `[FB]wg: ...` | 预执行的操作快照。 |
| stdout | `{COVERED} {gid1, gid2}` | 未过期双栏累计到第二次到达。 |
| stdout | `{TIMEOUT} {gid1, gid2}` | 双栏首个到达者等待超时。 |
| stdout | `{COVERED_OP} {pre, next}` | next BF 消费了 pre event。 |
| stdout | `{TIMEOUT_OP} {pre, next}` | next BF 未等到 pre event。 |

这些 `COVERED` 信号都不是 bug 结论；真正的缺陷线索仍是 panic 或 `WARNING: DATA RACE`。

## 超时与并发层次

当前存在多套互不联动的超时：

| 层次 | 当前值/来源 | 实际作用 |
|---|---|---|
| Executor 子进程 | `Config.TimeOut`；full 固定 30s，lite 取 CLI | `context.WithTimeout` 终止测试进程；0 会立即过期。 |
| Monitor watchdog | 固定 1 分钟 | worker 放弃等待本轮结果，但不会直接取消 Executor；Executor 仍由自身 context 管理。 |
| Goroutine barrier | 固定 10ms | `pointControl` 等待配对方。 |
| OP BF | 固定 `20s / 5 = 4s` | next 等待 pre event。 |
| Recover timeout | full 200s / lite CLI | 只写 `RECOVER_TIMEOUT` 环境变量，仓库内没有读取者。 |

Executor 也写入 `TIMEOUT` 环境变量，但 goroutine/sched runtime 没有读取它。不要把 CLI timeout 误认为 barrier 或 OP 等待时间。

并发层次也要区分：

- `full --max`：并行 Monitor（测试函数）数量，默认 24。
- `Monitor.Start`：无条件把 `cfg.MaxWorker` 改为 4，每个测试固定 4 个 Executor worker。
- 传入 Monitor 的 `ticket` channel 当前未使用。

## Monitor 终止与结果

每个 worker 执行：

```text
test binary -test.v -test.run <TestXxx_1>
```

Monitor 处理每份结果时：

1. 在 stdout/stderr 中搜索 `panic:`，在 stderr 中搜索 `WARNING: DATA RACE`，只写入 `LogCh`。
2. 解析 stderr 中的 pair、OP 和 `GORT_EDGE`，更新两个 corpus 与 goroutine 拓扑并集。
3. 检查是否切换阶段。
4. 解析 stdout，并在 `UseMutate` 开启时应用信号。
5. 新增 goroutine covered 时重置 `MaxQuit`；否则将其减 1。

`MaxQuit` 现在表示连续没有新增 goroutine covered 的预算；OP covered 或仅新增拓扑边不会重置它。`MaxExecution` 先从 `int` 转为 `int32`；默认的 10,000,000,000 会截断为 1,410,065,408，更大的配置还可能变成负数并立即退出。Monitor 使用 `etimes >= max`，未溢出时最多处理 `max` 份结果。

`SingleCrash`、`UseFeedBack`、`UseCoveredSched`、`UseStates`、`UseAnalysis`、`UseGuide`、`InitTurnCnt` 当前未被 Monitor 使用。`UseMutate=false` 只停止反馈应用，并不会停止取种子或运行时调度，因此会重复执行同一批种子。

`pkg/bug` 没有实现；panic/race 不进入 BugSet，Monitor 也不会返回 failure。分析实验结果必须以日志内容为准。

## Fixtures、脚本与产物

- `testdata/gobench/nonblocking_origin` 是未插桩对照；`nonblocking` 已保留原测试并新增 `TestXxx_1`。
- `testdata/myTest/corpus` 和 `corpus1` 是当前 goroutine/op 小样例；其他目录可能是旧 callstack 实验。
- `script/run_all.ps1` 不执行插桩。它对 7 个已插桩 GoBench 目录依次运行 `bins`，再运行 `full` 并把 stdout 重定向到 `zgortResult/<name>.txt`。
- 该脚本不生成 `raceResult`；其中内容是历史实验汇总。
- 结果目录、测试二进制和工具二进制均已纳入 Git，且 `.gitignore` 只忽略 IDE/编辑器文件。运行构建或实验前先检查 `git status`，不要意外提交大体积产物。

## 开发约定

### 修改插桩

新增 Pass 时：

1. 在 `pkg/inst/passes/` 实现 `InstPass` 的 `Before/GetPreApply/GetPostApply/After`。
2. 在 `cmd/inst/inst.go` 的两个 registry 分支中都注册，明确顺序依赖。
3. 检查表达式求值次数、defer 求值时机、select AST 位置和 import alias 冲突。
4. 为 Pass 增加最小输入到期望 AST/可编译输出测试；当前这层测试覆盖很弱。

### 修改协议或 Corpus

- stderr/stdout 文本是 runtime 与 `feedback` parser 的内部协议，修改生产端时必须同步修改解析器和 parser tests。
- `GORT_EDGE` 是逐行 JSON 协议；解析器会保留合法边并报告首个畸形协议行，Monitor 仍合并已成功解析的部分。
- `GortPhase` 跨 worker/Corpus 共享，继续使用 atomic；map/slice 访问保持在对应 `sync.RWMutex` 下。
- OP 的对象地址不能跨测试进程混合；改变预执行收集策略时必须先解决执行快照边界。
- `event.LoadAndDelete` 是一次性依赖消费；改变一对多语义时需要重新设计，而不是只调整 Corpus。

### 验证顺序

对普通核心代码变更，至少执行：

```shell
go test ./cmd ./pkg/...
```

涉及插桩或运行时调度时，还应在干净副本上完成 `inst -> bins -> lite/full` 集成验证，并分别检查：

- 插桩结果能通过编译和 `go test -race -c`。
- stderr 预执行协议可被 `ParseGortPairs` 解析。
- stdout 四种调度信号可被 `ParseSignals` 解析。
- race/panic 日志没有因只看最终 `PASS` 而被遗漏。
