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