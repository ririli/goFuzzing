# GoPie 项目指南

## 项目定位

GoPie 是 Go 并发缺陷实验的研究原型，通过 AST 插桩 + 调度搜索来发现数据竞争、panic 等并发 bug。不修改 Go runtime，不变异普通函数参数。


## AI 交互与修改原则

- 以代码实际行为为准；注释和字段名可能落后于实现，用中文进行回复
- 修改保持范围小并复用现有结构
- 插桩会原地覆盖源码且不幂等，不要对 `testdata/gobench/nonblocking_origin` 或未备份源码执行插桩
- 不要把 `bin/`、`testbins/`、`raceResult/`、`zgortResult/` 当作可靠构建产物

