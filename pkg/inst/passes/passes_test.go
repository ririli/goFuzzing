package passes

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"toolkit/pkg/inst"
)

// instrumentSource 将源码写入临时文件，依次执行给定 pass 后返回生成的代码文本。
func instrumentSource(t *testing.T, src string, passes ...inst.InstPass) string {
	t.Helper()
	dir := t.TempDir()
	file := filepath.Join(dir, "sample.go")
	if err := os.WriteFile(file, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	iCtx, err := inst.NewInstContext(file)
	if err != nil {
		t.Fatalf("NewInstContext() error = %v", err)
	}
	for _, p := range passes {
		inst.RunPass(p, iCtx)
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, iCtx.FS, iCtx.AstFile); err != nil {
		t.Fatalf("format.Node() error = %v", err)
	}
	return buf.String()
}

// ---------- WgPass：defer 分支禁止 WaitGroup 值拷贝 ----------

const wgSample = `package sample

import "sync"

type server struct {
	wg sync.WaitGroup
	pw *sync.WaitGroup
}

func (s *server) valueDefer() {
	defer s.wg.Done()
}

func (s *server) ptrDefer() {
	defer s.pw.Done()
}

func localDefer() {
	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Done()
	wg.Wait()
}
`

func TestWgPass_DeferDone_ValueReceiverTakesAddress(t *testing.T) {
	out := instrumentSource(t, wgSample, &WgPass{})

	// 值接收者必须取地址（_wg_N := &s.wg），禁止值拷贝：
	// 拷贝会引入 data race（拷贝读 vs 真 wg 的 Wait 写）并破坏计数器语义
	if !strings.Contains(out, "&s.wg") {
		t.Errorf("value receiver should be address-taken (&s.wg), got:\n%s", out)
	}
	if strings.Contains(out, "_wg_") && strings.Contains(out, ":= s.wg\n") {
		t.Errorf("value receiver must not be copied by value, got:\n%s", out)
	}
	// 局部变量同理取地址
	if !strings.Contains(out, "&wg") {
		t.Errorf("local value receiver should be address-taken (&wg), got:\n%s", out)
	}
}

func TestWgPass_DeferDone_PointerReceiverCopiesPointer(t *testing.T) {
	out := instrumentSource(t, wgSample, &WgPass{})

	if !strings.Contains(out, ":= s.pw") {
		t.Errorf("pointer receiver should be copied directly (:= s.pw), got:\n%s", out)
	}
	if strings.Contains(out, "&s.pw") {
		t.Errorf("pointer receiver must not be address-taken again, got:\n%s", out)
	}
}

func TestWgPass_DeferAdd_WrappedOnce(t *testing.T) {
	out := instrumentSource(t, wgSample, &WgPass{})

	// 共 4 处：valueDefer.Done / ptrDefer.Done / localDefer.Add / localDefer.Done
	// defer 语句被改写为包一层闭包（BF + 调用 + AF），普通语句前后插入 BF/AF
	if n := strings.Count(out, "InstWgBF"); n != 4 {
		t.Errorf("InstWgBF count = %d, want 4", n)
	}
	if n := strings.Count(out, "InstWgAF"); n != 4 {
		t.Errorf("InstWgAF count = %d, want 4", n)
	}
}

// ---------- GoroutinePass：方法接收者提前到父 goroutine 求值 ----------

const gortSample = `package sample

import "fmt"

type keeper struct{}

func (k *keeper) run(stop chan struct{}) { <-stop }

type tokenSimple struct {
	keeper *keeper
}

func (t *tokenSimple) enable(stop chan struct{}) {
	go t.keeper.run(stop)
}

func plain() {
	go fmt.Println("hello")
}
`

func TestGoroutinePass_MethodReceiverHoistedToParent(t *testing.T) {
	out := instrumentSource(t, gortSample, &GoroutinePass{})

	if n := strings.Count(out, "_recv_"); n == 0 {
		t.Fatalf("method receiver should be hoisted into a _recv_ temp, got:\n%s", out)
	}
	// 只有 go t.keeper.run 需要提取；go fmt.Println 的 fmt 是包名，不得提取
	if n := strings.Count(out, "_recv_"); n != 2 {
		// _recv_N 出现在赋值与调用两处，恰好 2 次；多于 2 次说明包调用被误提取
		t.Errorf("_recv_ occurrences = %d, want 2 (one assign + one use)", n)
	}
	if !strings.Contains(out, ":= t.keeper.run") || !strings.Contains(out, "(_arg_") {
		t.Errorf("inner call should use hoisted receiver and args, got:\n%s", out)
	}
}

func TestGoroutinePass_PackageQualifiedCallUntouched(t *testing.T) {
	out := instrumentSource(t, gortSample, &GoroutinePass{})

	// fmt.Println 必须原样保留在闭包内，不能被提取为临时变量
	if !strings.Contains(out, "fmt.Println(_arg_") && !strings.Contains(out, "fmt.Println(\"hello\")") {
		t.Errorf("package-qualified call should keep fmt selector, got:\n%s", out)
	}
}

// ---------- FunctionPass：函数体开头 //go: 指令不得被插入语句打断 ----------

const funcDirectiveSample = `package sample

type BeeMap struct{}

func (m *BeeMap) Delete(k interface{}) {
	//go:noinline
	m.lock()
}

func (m *BeeMap) lock() {}
`

func TestFunctionPass_GoDirectiveStaysBeforeInsertedStmts(t *testing.T) {
	out := instrumentSource(t, funcDirectiveSample, &FunctionPass{})

	// 旧缺陷：无位置节点导致保留的 //go: 指令被打进 selector 中间
	// （defer gopie_function.\n//go:noinline\nTrace(...)），指令失效并改变语义
	if strings.Contains(out, "gopie_function.\n") || strings.Contains(out, "gopie_function.\r\n") {
		t.Errorf("//go: directive split the selector (gopie_function.\\n...), got:\n%s", out)
	}
	// //go: 指令必须打印在插入的 PointControl/Trace 语句之前
	directiveIdx := strings.Index(out, "//go:noinline")
	pcIdx := strings.Index(out, "gopie_function.PointControl(")
	traceIdx := strings.Index(out, "gopie_function.Trace(")
	if directiveIdx == -1 {
		t.Fatalf("//go: directive lost after instrumentation, got:\n%s", out)
	}
	if pcIdx == -1 || traceIdx == -1 {
		t.Fatalf("inserted hooks missing, got:\n%s", out)
	}
	if directiveIdx > pcIdx || directiveIdx > traceIdx {
		t.Errorf("//go: directive should precede inserted hooks, got:\n%s", out)
	}
}
