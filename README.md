# GoPie

GoPie 是 Go 并发缺陷实验的研究原型，通过 AST 插桩 + 调度搜索来发现数据竞争、panic 等并发 bug。不修改 Go runtime，不变异普通函数参数。

## Usage

1. `GoPie` 基于 `Go 1.25.1` 实现，请先安装对应版本的 Go：

    https://go.dev/doc/install

2. 编译 `cmd` 下的二进制，编译完成后会生成两个可执行文件 `inst` 和 `fuzz`：

    ~~~shell
    go build -o ./bin ./cmd/...
    ~~~

3. 插桩待测项目并编译（插桩会原地覆盖源码，请先备份）：

    ~~~shell
    ./bin/fuzz --task inst --path your_project_to_be_tested
    ~~~

4. 编译测试二进制，产物将放在 `./testbins`：

    ~~~shell
    ./bin/fuzz --task bins --path your_project_to_be_tested --granularity 
    ~~~

5. 开始测试（可将输出重定向保存到文件）：

    ~~~shell
    ./bin/fuzz --task full --path path_of_test_binaries > xx.txt
    ~~~

## 颗粒度模式

`GoPie` 支持 goroutine 与 function 两种调度颗粒度：

- goroutine 模式：以 goroutine 并发对为调度单元，OP 对从已覆盖 goroutine 对的 gid 链路生成
- function 模式：以函数并发对为调度单元，OP 对从已覆盖函数对的 funcID 链路（[FB] 日志 fids 字段）生成

OP（操作对）仅在预执行种子收集结束后一次性生成，fuzzing 阶段再按覆盖信号增量补充。

### 切换方式

插桩与测试两个阶段都需要指定同一颗粒度（通过 `--granularity` 参数）：

~~~shell
# goroutine 模式（默认）
./bin/fuzz --task inst --path your_project_to_be_tested --granularity goroutine
./bin/fuzz --task full --path path_of_test_binaries --granularity goroutine

# function 模式
./bin/fuzz --task inst --path your_project_to_be_tested --granularity function
./bin/fuzz --task full --path path_of_test_binaries --granularity function
~~~

也可以改用 `FUZZ_MODE` 环境变量设置（CLI 参数优先于环境变量）：

~~~shell
FUZZ_MODE=function ./bin/fuzz --task inst --path your_project_to_be_tested
FUZZ_MODE=function ./bin/fuzz --task full --path path_of_test_binaries
~~~

参数大小写不敏感，支持 `function`/`func` 别名；非法值或未指定时回退为 `goroutine`。
