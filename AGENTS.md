# GoPie 项目指南

## 项目定位

GoPie 是 Go 并发缺陷实验的研究原型以及是我的课题（目标是发CCF-A会议），通过 AST 插桩 + 调度搜索来发现数据竞争、panic 等并发 bug。不修改 Go runtime，不变异普通函数参数。


## AI 交互与修改原则

- 以代码实际行为为准；注释和字段名可能落后于实现，用中文进行回复
- 修改保持范围小并复用现有结构
- 插桩会原地覆盖源码且不幂等，不要对 `testdata/gobench/nonblocking_origin` 或未备份源码执行插桩
- 不要把 `bin/`、`testbins/`、`raceResult/`、`zgortResult/` 当作可靠构建产物

## 跑实验指南

- 实验统一在 WSL 中进行，不要直接在 Windows 或 `/mnt/c`、`/mnt/d` 等挂载目录中运行。
- 开始实验前，先在 Windows 端的 GoPie 工作区检查代码改动，将本次实验需要的改动提交并 push 到当前远端分支；不要把无关改动一并提交。
- 随后在 WSL 中进入 `/home/riri/projects/gopie`，确认处于对应分支且工作区没有需要保留的未提交改动，再使用 `git pull --ff-only` 拉取 Windows 端刚刚 push 的最新版本。若存在本地改动、分支不一致或拉取冲突，先停止并处理，不要强制覆盖。
- GoPie 课题项目位于 `/home/riri/projects/gopie`；运行实验时使用 `scripts/` 中对应的 shell 脚本，不要手工拼接长命令。
- 被测项目统一放在 `/home/riri/realProjects/<项目目录>/` 下。每个项目准备三个相互独立的副本：原始名称、名称加 `G`、名称加 `F`。例如 Beego 的实际目录为 `BEEGO/beego`、`BEEGO/beegoG`、`BEEGO/beegoF`。
- 原始名称副本（如 `beego`）只用于原生 Go race detector 基线实验；`G` 副本用于 goroutine 颗粒度插桩与调度搜索；`F` 副本用于 function 颗粒度插桩与调度搜索。禁止跨副本复用已插桩源码。
- 对 `G`、`F` 副本进行插桩前，需要在被测项目的 `go.mod` 中加入 `replace`，将插桩代码引用的 `toolkit` 模块指向 WSL 本地的 `/home/riri/projects/gopie`。原始副本不需要该 `replace`；若被测项目包含多个独立 Go module，则每个会编译插桩代码的模块都要处理对应的 `go.mod`。
- 原生 race 基线使用 `bash scripts/run_project_allRaceTest.sh --project-path /home/riri/realProjects/<项目目录>/<原始名称>`；例如 Beego 使用 `/home/riri/realProjects/BEEGO/beego`。
- GoPie fuzz 实验使用 `bash scripts/run_full.sh`；需要并行启动 function/goroutine 两组长跑实验时，使用项目目录名运行 `bash scripts/do_run_full.sh <项目目录>`，例如 `bash scripts/do_run_full.sh BEEGO`。
- 完整实验顺序为：Windows 端提交并 push GoPie → WSL 端 `git pull --ff-only` 拉取最新 GoPie → 准备被测项目的三个干净副本 → 在 WSL 中编译 GoPie → 在原始副本上运行原生 race 基线 → 在 `G`、`F` 副本的相关 `go.mod` 中加入指向本地 GoPie 的 `toolkit` replace → 分别对 `G`、`F` 副本进行 goroutine、function 颗粒度插桩 → 分别生成测试二进制 → 使用 `scripts/do_run_full.sh <项目目录>` 并行长跑 → 检查运行日志和两个副本中的 `gopieRes`。
- 实验结果写入C:\Users\riri\Desktop\硕士课题（fuzzing）\真实项目实验结果\realProject.docx中。

## 实验时间标准

fuzz 与原生 race 基线统一按下述参数取值，beego、grpc 均按此口径跑，不要在单次实验里私自改数。

- fuzzing（`run_full.sh` / `do_run_full.sh`）：`--timeout 300`（单次调度执行的墙钟上限）、`--fuzz-time 1800`（单个测试函数整个搜索会话的上限）、`--recover-timeout 200`；并发固定为 4 个测试并行 × 每测试 4 子进程 = **16 个被测进程**（与 16 逻辑线程 1:1，`cmd/fuzz/full.go` 的 `max` 默认值），`MaxQuit=32`。
- 原生 race 基线（`run_project_allRaceTest.sh`）：`--timeout-minutes 5`（每包 `go test -race -timeout`）、`--count 1`。
- fuzz 结果在被测副本的 `gopieRes/`（`allpanic.txt`、`alldatarace.txt`、每二进制一份 txt）；race 基线结果在**执行脚本时所在目录**的 `race_results/<项目basename>_<时间戳>/`（`summary.txt`、`all_race_warnings.txt`、`all_panics.txt`、`package_logs/`），不在被测项目里，建议显式传 `--output-dir`。

### 时间控制点的边界（改标准前先读）

- 只有三层时间闸生效：单次执行 `--timeout`（`pkg/fuzzer/executor.go:64-65`）、单测试函数 `--fuzz-time`（`pkg/fuzzer/config.go:60` → `pkg/fuzzer/monitor.go:286`）、race 每包 `--timeout-minutes`。**项目级总时长和单二进制总时长没有任何上限**，`run_full.sh:86-110` 只是顺序遍历。
- `--recover-timeout` 目前是空转参数：`executor.go:92-93` 注入的 `RECOVER_TIMEOUT` 环境变量在全仓库没有读取点。插桩运行时真正生效的是硬编码值——`pkg/operation/operation.go:45-46`（5s / 1s）、barrier `BarrierTimeout=10ms`（`pkg/goroutine/env.go:25`、`pkg/function/env.go:25`）。
- 单二进制耗时由并发度决定而非时间参数：`cmd/fuzz/full.go` 的 `max`（同时 fuzz 的测试函数数，默认 4）× `MaxWorker=4`（`full.go:72` 硬编码）= 被测进程总数；`--max` 未被 `run_full.sh` 透传。**并发超卖会污染结论**：barrier 等待窗口硬编码 10ms，进程排队导致对端迟迟不到，pair 连续 5 次超时即被永久标为 infeasible 并从候选集删除（`corpus_gort.go:478-487`、`corpus_func.go:458-464`、`corpus_op.go:402-408`），搜索空间被提前砍小。
- 收敛还受非时间闸门影响：`MaxExecution=250`、`MaxQuit=32`（`full.go:77-78`）、`MaxPreExecRound=30`（`config.go:92`）。注意 `MaxQuit` 的「进展」判据是 `newOracleFinding || newEdges>0 || InPreExec()`（`monitor.go:343`），function 模式拓扑边极多、几乎每轮都有新边，因此 `MaxQuit` 很难触发、单个测试通常跑到 1800s 上限才停；`SingleCrash` 在 `full` 路径下为 false（`config.go:79`），发现 bug 后不会提前收尾。
- beego 的历史数据是在 `max=12`（48 进程）+ `MaxQuit=200` 下跑出来的，与新口径不同；跨项目对比要么重跑 beego，要么在结果表里分栏注明。

### 两套口径不可直接相减的地方

- fuzz 的 1800s 是**每个测试函数独享**，race 的 5min 是**每个包内所有测试共享**，测试多的包（如 grpc `test/` 有 315 个）在 race 侧会大面积超时。
- race 计数用 `grep -c 'WARNING: DATA RACE'`（`run_project_allRaceTest.sh:148`），同一竞争重复计数；fuzz 按 `signature` 去重。对比效率请统一换算成「每 1000 CPU 秒的去重检出数」，不要用墙钟和原始行数。
- `run_project_allRaceTest.sh:224-226` 在发现任何 race/panic 或包失败时 `exit 1`，不能拿返回码当运行健康判据。
