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

### 串行重跑实验（serial chain，2026-09-30 起）

重跑全项目集时用这套脚本，不要再手工拼 `do_run_full.sh` 的双组并行：

- `scripts/serial_queue.sh` 是唯一的队列定义：顺序 `websocket→gin→gorums→fiber→beego→etcd→prometheus→grpc`，每个项目先 `G`（goroutine）后 `F`（function），共 16 步；小项目在前是因为 function 模式实测慢 30~78 倍，长尾要留到最后。改顺序/改项目集只改这里。
- `scripts/prepare_copy.sh` 重建单个副本：把旧 `G`/`F` 副本 `mv` 到 `/home/riri/retired/` → 从原始副本 `cp -a` → 移走无测试包的嵌套 module → 移走 `go.work` → 每个 `go.mod` 先加 `replace` → `bin/fuzz --task inst` → 再加 `require toolkit v0.0.0` → `go mod tidy` → `bin/fuzz --task bins`。**脚本里不允许出现 `rm -rf`**，所有「删」都是可回滚的 mv；旧 `testbins/<step>` 也会先 mv 走，防止陈旧二进制混入长跑。
- **副本里不能留 `go.work`**（2026-10-02 踩坑）：gorums、prometheus 的仓库根带 `go.work`，其 `use` 条目正好是 `PRUNE_DIRS` 要剪掉的 module（`examples`、`compliance`、`internal/tools` 等）。剪完 workspace 指向不存在的目录，Go 在 workspace 模式下**连根 module 都编译不了**，gorumsG 首轮 12/12 目录全挂在 `cannot load module examples listed in go.work file`，产出 0 个测试二进制并 HALTED。处理方式是 mv 留档而非改写 `use` 列表：接线模型本来就是「每个 `go.mod` 各自 replace/require」，根 `go.mod` 并不 require 被剪的 module，其余 6 个项目也都没有 go.work。后来上队列的 prometheus、以及任何新引入的带 workspace 的项目，都靠这一步兜住。
- `scripts/serial_chain.sh` 是常驻状态机，串行推队列，一步跑完才进下一步；`state/<step>.prepared` 表示已插桩+已编译（插桩不幂等，靠它避免二次覆盖），`state/<step>.done` 表示该步长跑跑满全部二进制；二进制级别由 `run_full.sh --resume` 的 `gopieRes/.done/` 标记续跑。任一步失败就写 `HALTED` 并退出，不自动往下跑，等人工判断。
- `scripts/serial_tick.sh` 给小时级自动化调用：查活、必要时用 `systemd-run --user` 把驱动拉回来、打印 `STATUS`；`--report-only` 只看状态不动手。
- **`systemd-run --user` 不继承调用方环境变量**（2026-09-30 实测踩坑：沙箱自测的 tick 拉起驱动时丢掉了我 `export` 的根路径，驱动静默回落到 `/home/riri/realProjects`，把真实的 `GORILLA/websocketG` 重新插桩了一遍）。因此根路径一律经 `require_roots` 校验：要么四个根全是真实默认值，要么显式 `GOPIE_SANDBOX=/tmp/xxx` 让四个根一起派生且必须在 `/tmp` 下；只改单个根的混合状态直接拒绝运行。拉起时还要用 `--setenv` 把根路径显式传进单元。

## 实验时间标准

fuzz 与原生 race 基线统一按下述参数取值，beego、grpc 均按此口径跑，不要在单次实验里私自改数。

- fuzzing（`run_full.sh` / `do_run_full.sh`）：`--timeout 300`（单次调度执行的墙钟上限）、`--fuzz-time 1800`（单个测试函数整个搜索会话的上限）、`--recover-timeout 200`；并发固定为 4 个测试并行 × 每测试 4 子进程 = **16 个被测进程**（与 16 逻辑线程 1:1，`cmd/fuzz/full.go` 的 `max` 默认值），`MaxQuit=32`。
- 原生 race 基线（`run_project_allRaceTest.sh`）：`--count 30`（每包重复 30 轮，提高竞争命中概率）、`--timeout-minutes 25`（每包 `go test -race -timeout`）。
- fuzz 结果在被测副本的 `gopieRes/`（`allpanic.txt`、`alldatarace.txt`、每二进制一份 txt）；race 基线结果在**执行脚本时所在目录**的 `race_results/<项目basename>_<时间戳>/`（`summary.txt`、`all_race_warnings.txt`、`all_panics.txt`、`package_logs/`），不在被测项目里，建议显式传 `--output-dir`。

### race 基线产物分工（看结果时先读这条）

`go test -json` 的输出是一行一条 `{"Output":"..."}`，直接当报告读等于读 json。各文件职责：

- `package_logs/*.json` —— **一手证据**，逐包原始 `-json` 流，只在需要还原上下文/重算统计时用，别直接 grep 看。
- `REPORT.md` —— **给人读/进论文的报告**（`scripts/race_digest.py` 生成）：按「访问点对」去重的数据竞争清单（含测试名归属、解码过的栈）、真实 panic（自动剔除 `panic: test timed out` 假 panic）、未跑满 `--count` 轮次的欠采样包清单、非干净包表格。
- `all_race_warnings.txt` / `all_panics.txt` —— 只含命中的块，且已由 `scripts/decode_test_json.py` 解码成纯文本。
- `summary.txt`（原口径不变）与 `summary.csv`（逐包一行，方便贴表）。
- 重跑历史数据不必再跑实验：`python3 scripts/race_digest.py <结果目录> <项目名片段>` 即可从 `package_logs` 回填 `REPORT.md`。

### 时间控制点的边界（改标准前先读）

- 只有三层时间闸生效：单次执行 `--timeout`（`pkg/fuzzer/executor.go:64-65`）、单测试函数 `--fuzz-time`（`pkg/fuzzer/config.go:60` → `pkg/fuzzer/monitor.go:286`）、race 每包 `--timeout-minutes`。**项目级总时长和单二进制总时长没有任何上限**，`run_full.sh:86-110` 只是顺序遍历。
- `--recover-timeout` 目前是空转参数：`executor.go:92-93` 注入的 `RECOVER_TIMEOUT` 环境变量在全仓库没有读取点。插桩运行时真正生效的是硬编码值——`pkg/operation/operation.go:45-46`（5s / 1s）、barrier `BarrierTimeout=10ms`（`pkg/goroutine/env.go:25`、`pkg/function/env.go:25`）。
- 单二进制耗时由并发度决定而非时间参数：`cmd/fuzz/full.go` 的 `max`（同时 fuzz 的测试函数数，默认 4）× `MaxWorker=4`（`full.go:72` 硬编码）= 被测进程总数；`--max` 未被 `run_full.sh` 透传。**并发超卖会污染结论**：barrier 等待窗口硬编码 10ms，进程排队导致对端迟迟不到，pair 连续 5 次超时即被永久标为 infeasible 并从候选集删除（`corpus_gort.go:478-487`、`corpus_func.go:458-464`、`corpus_op.go:402-408`），搜索空间被提前砍小。
- 收敛还受非时间闸门影响：`MaxExecution=250`、`MaxQuit=32`（`full.go:77-78`）、`MaxPreExecRound=30`（`config.go:92`）。注意 `MaxQuit` 的「进展」判据是 `newOracleFinding || newEdges>0 || InPreExec()`（`monitor.go:343`），function 模式拓扑边极多、几乎每轮都有新边，因此 `MaxQuit` 很难触发、单个测试通常跑到 1800s 上限才停；`SingleCrash` 在 `full` 路径下为 false（`config.go:79`），发现 bug 后不会提前收尾。
- `--count` 与 `--timeout-minutes` 必须成对放大：`-timeout` 是**整包 30 轮共享**的闸，按单轮耗时 ×30 估算。fabio 实测单轮 `config` 40s、`cert` 35s、`admin` 19s，若沿用 5min 会在 30 轮中途被杀，且超时产生的 `panic: test timed out after Xm0s` 会被 `run_project_allRaceTest.sh:149` 的 panic 正则计入，凭空造出假 panic。25min 是给单轮 ≤50s 的包留余量。
- c30 口径实测成本（单线程串行、WSL 16 线程机器）：beego 53 包 **112min**、fabio 16 包 **38min**、grpc 151 包 **147min**（以上 2026-09-27~28 首跑）；etcd 94 包 / 12 module **178min**、websocket 1 包 **52s**（2026-09-29 首跑）。规划新项目预算按这个量级估，多 module 项目的集成测试（etcd 的 `tests/v3/integration*`）是主要成本来源。
- **墙钟对构建缓存极敏感，别直接跨项目比**：beego 同一份代码同一口径，09-27 首跑 112min、09-29 复跑 60.5min（约 2×），而检出数逐条相同（7 对去重竞争、7 条真实 panic）。新项目首跑一律是冷缓存，墙钟里含大量编译时间；要用「每 1000 CPU 秒去重检出数」就得统一注明缓存状态，或先预热再计时。
- 25min 闸仍会截断慢包：beego 的 `task`、`client/httplib/mock`、`server/web/session/ssdb`，etcd 的 `tests/v3/integration`、`integration/clientv3`、`integration/clientv3/connectivity`、`integration/clientv3/lease`（后 4 个各跑满 26min 仍未完成 30 轮）都没跑满，检出数属**欠采样**，成表时要注明；截断还会附带假 panic，统计真实 panic 前先剔除 `panic: test timed out`。
- **已定口径：被 25min 闸截断的包不放大 timeout 重跑**（2026-09-29 决定）。为几个包单开 60min 闸会破坏「全项目统一 c30/25min」的可比性，成本也不成比例；处理方式是成表时标欠采样或直接排除，并在正文写明这些包的 0 检出不能作为「无竞争」的证据。
- grpc 原始副本按整份（7 个 module、151 个含测试包）跑，而 fuzz 侧只在根模块插桩；对比时取 summary 表里的根模块行，别把嵌套 module 的成本算进同一栏。

- beego 的历史数据是在 `max=12`（48 进程）+ `MaxQuit=200` 下跑出来的，与新口径不同；跨项目对比要么重跑 beego，要么在结果表里分栏注明。

### 两套口径不可直接相减的地方

- fuzz 的 1800s 是**每个测试函数独享**，race 的 25min 是**每个包内 30 轮重复共享**，测试多的包（如 grpc `test/` 有 315 个）在 race 侧会大面积超时。
- race 计数用 `grep -c 'WARNING: DATA RACE'`（`run_project_allRaceTest.sh:148`），同一竞争重复计数；fuzz 按 `signature` 去重。对比效率请统一换算成「每 1000 CPU 秒的去重检出数」，不要用墙钟和原始行数。
- `run_project_allRaceTest.sh:224-226` 在发现任何 race/panic 或包失败时 `exit 1`，不能拿返回码当运行健康判据。
