# GoPie 项目指南

## 项目定位

GoPie 是 Go 并发缺陷实验的研究原型，通过 AST 插桩 + 调度搜索来发现数据竞争、panic 等并发 bug。不修改 Go runtime，不变异普通函数参数。

主链路：**插桩源码 → 编译 -race 测试二进制 → 预执行收集种子 → 调度搜索（Fuzzing）→ 解析 race/panic/fatal 输出**。

> Monitor 将每次执行输出交给 `pkg/bug` 按稳定签名去重 race/panic/fatal，据此返回 `FAIL`。`PASS` 只表示本次未解析到触发型 oracle。

---

## 完整 Fuzzing 流程

### 1. 插桩：`fuzz --task inst --path <dir>`

并发调用独立 `inst` 二进制，对每个 `.go` 文件运行 5 个 Pass（**gort → channel → select → waitgroup → test**），原地覆盖源码。**插桩不幂等**。

### 2. 编译：`fuzz --task bins --path <dir> -o testbins/local`

对每个含 `.go` 的目录并发执行 `go test -race -c`，输出测试二进制。用 `-test.list _1` 枚举 `Test*_1` 测试函数。

### 3. 调度搜索：`fuzz --task full --path testbins/local`

遍历二进制，为每个 `Test*_1` 启动 Monitor（`--max` 默认 24 并发），每个 Monitor 固定 4 个 Executor worker。

`fuzz --task lite` 是单测试简化版。

---

## Monitor 内部：两阶段 Fuzzing 循环

每个 Monitor 持有独立的 `CorpusGort`、`CorpusOp` 和 `bug.Set`。

```
for each execution:
  corpusGort.Get() + corpusOp.Get() → Enter/Op pair
  Executor.Run() → stdout + stderr
  ParseSignals(stdout) + bug.Parse(stdout, stderr)
  ParseGortPairs(stderr) + ParseGortEdges(stderr) → 更新 Corpus
  TryEndPreExec → 判断阶段切换
  ApplySignals → 应用反馈，刷新 TryPairs
```

### Phase 0：预执行（种子收集）

- `Input` / `InputOp` 为空，不设 `RECORD_STACK` / `SCHED_DEBUG`
- 收集 goroutine 生命周期、父子关系、`[FB]` 操作日志
- 输出 `[GORT_EDGE]`（拓扑边）、`[COVERED]`（直接重叠）、`[SUSPECT]`（结构推断）
- Goroutine 对推断：直接重叠（1.0）> 纯结构兄弟（0.5）> 邻接父子/兄弟（0.3-0.5）
- 连续 3 轮无新对或达到 `MaxPreExecRound`（默认 30）→ 切换到 Phase 1
- 切换后从 SusConPairs 选种填充 TryPairs，并生成危险操作对

### Phase 1：调度搜索

- 设置 `RECORD_STACK=1`（关闭生命周期记录）和 `SCHED_DEBUG=1`（关闭 FB 日志）
- `Input=(gid1,gid2)...` 下发 goroutine 对，`InputOp=(preId,nextId)...` 下发操作对
- Goroutine 双栏调度：两个 goroutine 都到达 `Enter` 时放行并输出 `{COVERED}`，10ms 超时输出 `{TIMEOUT}`
- OP 调度：next 的 BF 等待 pre 的 AF 完成（`event.LoadAndDelete`），成功输出 `{COVERED_OP}`，4s 超时输出 `{TIMEOUT_OP}`
- 新 covered 对触发邻接推断（向 parent/child/sibling 扩展一跳）
- 选种分数：goroutine = `confidence*10 - timeoutCount*2`，OP = `5 - timeoutCount*2`
- 初始 `selectNum=1`，无 covered 但有 timeout 时翻倍（上限 64），累计 5 次 timeout 移入 InfeasiblePairs

### Monitor 终止

- `etimes >= MaxExecution`（默认 10,000,000,000，int32 截断为 1,410,065,408）
- 连续 `MaxQuit` 次无新 goroutine covered 且无新 oracle finding
- `SingleCrash && triggered`：首次 race/panic/fatal 立即退出
- 返回 `(true, ["FAIL", summary])` 或 `(false, ["PASS", ""])`

---

## 技术基线

- 模块名：`toolkit`，Go 1.19
- 直接依赖：`github.com/jessevdk/go-flags v1.5.0`、`golang.org/x/tools v0.2.0`

---

## 目录与职责

```
gopie/
├── cmd/
│   ├── fuzz/           # --task 路由：inst / bins / full / lite
│   ├── inst/inst.go    # 独立 AST 插桩器
│   ├── flags.go        # CLI flags
│   └── utils.go        # 文件枚举、AST 写回
├── pkg/
│   ├── inst/           # InstContext、Pass registry、执行和写回
│   │   └── passes/     # gort / channel / select / waitgroup / test
│   ├── goroutine/      # 生命周期追踪、双栏调度、并发对推断
│   ├── sched/          # 操作日志、pre→next 操作调度
│   ├── feedback/       # stderr/stdout 协议解析
│   ├── fuzzer/         # Config、Executor、Monitor、两个 Corpus
│   ├── bug/            # race/panic/fatal oracle 解析、签名去重
│   └── utils/          # gofmt / hash 辅助
├── testdata/
│   ├── gobench/nonblocking_origin/  # 35 个未插桩对照案例
│   └── gobench/nonblocking/         # 对应的 35 个已插桩案例
├── bin/                # Windows x86-64 PE 工具
├── testbins/           # Windows x86-64 PE 测试二进制
├── script/run_all.ps1  # 7 组 GoBench 的 bins + full 脚本
├── raceResult/         # 历史汇总
└── zgortResult/        # 历史 full 输出
```

---

## 构建与验证

```shell
go build -o ./bin ./cmd/...
go test ./cmd ./pkg/...
```

不要用 `go test ./...`（存在 vet 错误、硬编码路径、缺失测试等问题）。

---

## AST 插桩要点

### Pass 覆盖范围

| Pass | 注入行为 | 限制 |
|------|---------|------|
| GoroutinePass | 包装 `go f(args)`，子 goroutine 调用 `Enter/Exit` | `CurrentGid()` 在父 goroutine 求值，参数移入子 goroutine 可能改变语义 |
| ChRecPass | 普通 send/close 前后插 hook | 不处理 receive/make；AF 会再次求值 channel 表达式 |
| SelectPass | 只在 select send body 开头插 `InstChSelectAF` | 只有 AF，该操作只能做 pre 不能做 next |
| WgPass | Add/Done 前后插 hook | 不处理 Wait；可能误插自定义方法 |
| TestPass | 生成 `TestXxx_1`，注入 main 生命周期 hook | 浅复制原测试体，不是调用原测试 |

defer close/WaitGroup 改为 defer 闭包，receiver/参数从注册时求值变为执行时求值，可能改变语义。

### 静态 ID

每文件 `uint64(hash.Hash32(filePath)) << 32` + `atomic.AddUint64` 递增。ID 依赖路径字符串，不保证绝对无碰撞。`CallLoc` 当前输出 `:0`。

### 写回与幂等性

- 原地覆盖，不排除 vendor/generated/origin/已插桩文件
- **再次插桩会重复包装**，不幂等
- 写回后仅 `go fmt` 检查，不能发现类型错误
- `fuzz inst` 结果 channel 容量 100，文件数约 117+ 可能死锁

---

## 输出协议

| 流 | 格式 | 含义 |
|------|------|------|
| stderr | `[COVERED] gid1,gid2\|loc1,loc2\|1.00\|observed;` | 生命周期区间直接重叠 |
| stderr | `[SUSPECT] gid1,gid2\|loc1,loc2\|conf\|source;` | 结构推断的 goroutine 对 |
| stderr | `[GORT_EDGE] {"parent":p,"child":c,"count":n}` | 静态父子边和动态出现次数 |
| stderr | `[FB]chan: ...` / `[FB]wg: ...` | 预执行操作快照 |
| stdout | `{COVERED} {gid1, gid2}` | 双栏配对成功 |
| stdout | `{TIMEOUT} {gid1, gid2}` | 双栏等待超时 |
| stdout | `{COVERED_OP} {pre, next}` | OP 调度配对成功 |
| stdout | `{TIMEOUT_OP} {pre, next}` | OP 调度等待超时 |

---

## 超时层次

| 层次 | 值 | 作用 |
|------|-----|------|
| Executor 子进程 | full 30s / lite 取 CLI | `context.WithTimeout` 终止测试进程 |
| Monitor watchdog | 1 分钟 | worker 放弃等待本轮结果 |
| Goroutine barrier | 10ms | `pointControl` 等待配对方 |
| OP BF | 4s（20s/5） | next 等待 pre event |
| Recover timeout | full 200s | 仅写环境变量，无读取者 |

---

## AI 交互与修改原则

- 以代码实际行为为准；注释和字段名可能落后于实现
- 修改保持范围小并复用现有结构
- 插桩会原地覆盖源码且不幂等，不要对 `testdata/gobench/nonblocking_origin` 或未备份源码执行插桩
- 不要把 `bin/`、`testbins/`、`raceResult/`、`zgortResult/` 当作可靠构建产物

### 修改插桩

1. 在 `pkg/inst/passes/` 实现 `InstPass` 接口
2. 在 `cmd/inst/inst.go` 两个 registry 分支注册，明确顺序依赖
3. 检查表达式求值次数、defer 求值时机和 import alias 冲突

### 修改协议或 Corpus

- stderr/stdout 文本修改时须同步更新 `feedback` parser
- `GortPhase` 用 atomic；map/slice 访问保持 `sync.RWMutex`
- OP 对象地址不能跨进程混合
- `event.LoadAndDelete` 是一次性依赖消费

### 验证

```shell
go test ./cmd ./pkg/...
```

涉及插桩或调度时，在干净副本完成 `inst → bins → lite/full` 集成验证。
