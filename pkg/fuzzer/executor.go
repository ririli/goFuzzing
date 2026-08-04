package fuzzer

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
	"toolkit/pkg/feedback"
)

type Executor struct {
}

type Output struct {
	Err   error
	O     string
	Trace string
	Time  time.Duration
}

type Input struct {
	cmd            string
	args           []string
	timeout        int
	recovertimeout int
	//
	pairInput *PairInput              // 统一的并发对输入（goroutine或函数模式）
	gortPair  *feedback.InputGortPair // 已废弃，保留用于兼容
	funcPair  *feedback.InputPair     // 已废弃，保留用于兼容
	tryOpPair *feedback.InputOpPair
}

// 复用缓冲区的全局池（按需调整初始容量）
var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 4096)) // 初始容量 4KB
	},
}

func (e *Executor) Run(in Input) Output {
	// 1. 从池中获取缓冲区（用于小数据场景）
	stdoutBuf := bufferPool.Get().(*bytes.Buffer)
	stderrBuf := bufferPool.Get().(*bytes.Buffer)
	defer func() {
		// 重置并放回池中
		stdoutBuf.Reset()
		stderrBuf.Reset()
		bufferPool.Put(stdoutBuf)
		bufferPool.Put(stderrBuf)
	}()

	// 2. 创建带有超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(in.timeout)*time.Second)
	defer cancel()

	// 3. 执行命令并绑定上下文
	command := exec.CommandContext(ctx, in.cmd, in.args...)

	var strPair string
	if in.pairInput != nil && !in.pairInput.IsEmpty() {
		strPair = "Input=" + in.pairInput.ToString()
	} else {
		strPair = "Input="
	}
	var strOpPair string
	if in.tryOpPair == nil {
		strOpPair = "InputOp="
	} else {
		strOpPair = "InputOp=" + in.tryOpPair.ToString()
	}
	fmt.Println("=====strPair====")
	fmt.Println(strPair)
	fmt.Println(strOpPair)
	command.Env = append(os.Environ(), strPair, strOpPair)
	if in.timeout != 0 {
		command.Env = append(command.Env, fmt.Sprintf("TIMEOUT=%v", in.timeout))
	}
	if in.recovertimeout != 0 {
		command.Env = append(command.Env, fmt.Sprintf("RECOVER_TIMEOUT=%v", in.recovertimeout))
	}
	// 传递是否记录调用栈的标志
	if in.pairInput != nil && !in.pairInput.IsEmpty() {
		command.Env = append(command.Env, "RECORD_STACK=1")
		command.Env = append(command.Env, "SCHED_DEBUG=1")
	}
	// 4. 使用管道流式读取输出（避免全量加载）
	stdoutPipe, _ := command.StdoutPipe()
	stderrPipe, _ := command.StderrPipe()

	// 5. 启动异步流式处理
	stdoutDone := make(chan struct{})
	stderrDone := make(chan struct{})

	// 流式处理标准输出（按行处理）
	go streamProcess(stdoutPipe, stdoutBuf, stdoutDone)
	go streamProcess(stderrPipe, stderrBuf, stderrDone)

	// 6. 启动命令并等待结束
	start := time.Now()
	err := command.Run()

	// 7. 等待流式处理完成
	<-stdoutDone
	<-stderrDone

	// 8. 如果执行因超时被杀，追加标记
	if ctx.Err() == context.DeadlineExceeded {
		fmt.Fprintf(stdoutBuf, "{TIMEOUT_EXEC} execution timed out after %ds\n", in.timeout)
	}

	stdoutContent := stdoutBuf.String()
	stderrContent := stderrBuf.String()
	return Output{
		Err:   err,
		O:     stdoutContent,
		Trace: stderrContent,
		Time:  time.Since(start),
	}
}

func streamProcess(reader io.Reader, buf *bytes.Buffer, done chan struct{}) {
	defer close(done)
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		buf.Write(scanner.Bytes())
		buf.WriteByte('\n')
	}
}
