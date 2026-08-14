// 插桩语义修复验证文件
// 覆盖 Issue 4.1, 4.2, 4.3, 4.4 四个场景
// 运行: ./bin/fuzz --task inst --path testbins/myTest/instTest
package instTest

import (
	"sync"
	"testing"
	goroutine "toolkit/pkg/goroutine"
	sched "toolkit/pkg/operation"
)

// ============================================================
// 辅助函数
// ============================================================

func getChan() chan int { return make(chan int, 1) }

func getValue() int { return 42 }

type Wrapper struct {
	Ch0 chan int
	Ch1 chan int
}

func (w *Wrapper) getChan2() chan int { return w.Ch1 }

var chMap = map[string]chan int{"a": make(chan int, 1)}

func foo(a int) {}

// ============================================================
// Issue 4.1: goroutine 参数求值位置
//
// go f(expr) 中的 expr 应该在父 goroutine 求值。
// 插桩前 bug: 整个 f(expr) 被搬进子 goroutine 闭包内部。
// 修复后: expr 先求值为 _arg_N_X 临时变量，再传入闭包。
// ============================================================

func TestGortArgEval(t *testing.T) {
	_arg_4200315755918524417_0 :=
		// 4.1a: 函数调用作参数 —— getValue() 必须在父 goroutine 求值
		getValue()
	go func(_parentGid uint64) {
		goroutine.

			// 4.1b: 字面量作参数 —— 无副作用，但验证临时变量机制
			Enter(4200315755918524417, _parentGid)
		defer goroutine.Exit(4200315755918524417)
		println(_arg_4200315755918524417_0)
	}(goroutine.CurrentGid())
	_arg_4200315755918524418_0 := "literal_arg"
	go func(_parentGid uint64) {
		goroutine.

			// 4.1c: 多参数 + 混合 —— 第一个是函数调用，第二个是局部变量
			Enter(4200315755918524418, _parentGid)
		defer goroutine.Exit(4200315755918524418)
		println(_arg_4200315755918524418_0)
	}(goroutine.CurrentGid())

	local := 100
	_arg_4200315755918524419_0 := getValue() + local
	go func(_parentGid uint64) {
		goroutine.

			// 4.1d: 复杂表达式作参数
			Enter(4200315755918524419, _parentGid)
		defer goroutine.Exit(4200315755918524419)
		foo(_arg_4200315755918524419_0)
	}(goroutine.CurrentGid())
	_arg_4200315755918524420_0 := len([]int{1, 2, 3})
	go func(_parentGid uint64) {
		goroutine.

			// ============================================================
			// Issue 4.2: defer 闭包求值时机
			//
			// defer close(ch) 和 defer wg.Done() 中的变量应在 defer 注册时求值。
			// 插桩前 bug: 原 defer 被替换为 defer func(){...}()，变量延迟到执行时求值。
			// 修复后: 变量先求值为临时变量，再被 defer 闭包使用。
			// ============================================================
			Enter(4200315755918524420, _parentGid)
		defer goroutine.Exit(4200315755918524420)
		foo(_arg_4200315755918524420_0)
	}(goroutine.CurrentGid())
}

func TestDeferEval(t *testing.T) {
	// 4.2a: defer close(ch) —— ch 应在注册时求值
	ch1 := make(chan int, 1)
	_ch_4200315755918524421 := ch1
	defer

	// 4.2b: defer wg.Done() —— wg 应在注册时求值
	func() {
		sched.InstChBF(4200315755918524421)
		close(_ch_4200315755918524421)
		sched.InstChAF(4200315755918524421, _ch_4200315755918524421,

			// 4.2c: defer wg.Add() with negative delta (模拟)
			"close")
	}()

	var wg sync.WaitGroup
	sched.InstWgBF(4200315755918524430)
	wg.Add(1)
	sched.InstWgAF(4200315755918524430, &wg, "add")
	_wg_4200315755918524431 := wg
	defer func() {
		sched.InstWgBF(4200315755918524431)
		_wg_4200315755918524431.Done()
		sched.InstWgAF(4200315755918524431, &_wg_4200315755918524431, "done")
	}()

	var wg2 sync.WaitGroup
	sched.InstWgBF(4200315755918524432)
	wg2.Add(1)
	sched.InstWgAF(4200315755918524432,

		// ============================================================
		// Issue 4.3: channel 双重求值
		//
		// getChan() <- v 中的 getChan() 不应该被求值两次
		// (一次在原生 send，一次在 AF hook)。
		// 修复后: channel 表达式先求值为 _ch_N 临时变量，原生 send 和 AF 共用。
		// ============================================================
		&wg2, "add")
	_wg_4200315755918524433 := wg2
	defer func() {
		sched.InstWgBF(4200315755918524433)
		_wg_4200315755918524433.Add(-1)
		sched.InstWgAF(4200315755918524433, &_wg_4200315755918524433, "add")
	}()
}

func TestChannelEval(t *testing.T) {
	_ch_4200315755918524422 :=
		// 4.3a: 函数返回 channel —— getChan() 不应被调用两次
		getChan()
	sched.InstChBF(

		// 4.3b: 字段访问 —— 本身无副作用，但验证临时变量
		4200315755918524422)
	_ch_4200315755918524422 <- 42
	sched.InstChAF(4200315755918524422, _ch_4200315755918524422, "send")

	w := &Wrapper{Ch0: make(chan int, 1)}
	_ch_4200315755918524423 := w.Ch0
	sched.InstChBF(

		// 4.3c: 方法调用返回 channel
		4200315755918524423)
	_ch_4200315755918524423 <- 100
	sched.InstChAF(4200315755918524423, _ch_4200315755918524423, "send")

	// ============================================================
	// Issue 4.4: close 不支持复杂表达式
	//
	// 插桩前 bug: close(m["key"])、close(obj.field) 等非简单标识符被静默跳过。
	// 修复后: 任何表达式都可以，先求值为临时变量再 close。
	// ============================================================
	_ch_4200315755918524424 := w.getChan2()
	sched.InstChBF(4200315755918524424)
	_ch_4200315755918524424 <- 20
	sched.InstChAF(4200315755918524424, _ch_4200315755918524424, "send")
}

func TestCloseExpr(t *testing.T) {
	_ch_4200315755918524425 :=
		// 4.4a: close(map[index]) —— 之前会被跳过
		chMap["a"]
	sched.

		// 4.4b: close(obj.field) —— 之前会被跳过
		InstChBF(4200315755918524425)
	close(_ch_4200315755918524425)
	sched.InstChAF(4200315755918524425, _ch_4200315755918524425, "close")

	w := &Wrapper{Ch0: make(chan int, 1)}
	_ch_4200315755918524426 := w.Ch0
	sched.

		// 4.4c: close(函数返回的 channel) —— 之前会被跳过
		InstChBF(4200315755918524426)
	close(_ch_4200315755918524426)
	sched.InstChAF(4200315755918524426, _ch_4200315755918524426, "close")
	_ch_4200315755918524427 := getChan()
	sched.

		// 4.4d: 简单的 close(local) —— 之前就能处理，验证不受影响
		InstChBF(4200315755918524427)
	close(_ch_4200315755918524427)
	sched.InstChAF(4200315755918524427, _ch_4200315755918524427, "close")

	ch := make(chan int, 1)
	_ch_4200315755918524428 := ch
	sched.

		// 4.4e: defer close(复杂表达式)
		InstChBF(4200315755918524428)
	close(_ch_4200315755918524428)
	sched.InstChAF(4200315755918524428, _ch_4200315755918524428, "close")

	ch2 := make(chan int, 1)
	_ch_4200315755918524429 := ch2
	defer func

	// ============================================================
	// TestXxx_1 入口 —— TestPass 需要此函数来注入 main hook
	// ============================================================
	() {
		sched.InstChBF(4200315755918524429)
		close(_ch_4200315755918524429)
		sched.InstChAF(4200315755918524429, _ch_4200315755918524429, "close")
	}()
}

func TestSemantic_1(t *testing.T) {
	t.Run("GortArgEval", TestGortArgEval)
	t.Run("DeferEval", TestDeferEval)
	t.Run("ChannelEval", TestChannelEval)
	t.Run("CloseExpr", TestCloseExpr)
}
func TestGortArgEval_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	_arg_4200315755918524417_0 := getValue()
	go func(_parentGid uint64) {
		goroutine.Enter(4200315755918524417, _parentGid)
		defer goroutine.Exit(4200315755918524417)
		println(_arg_4200315755918524417_0)
	}(goroutine.CurrentGid())
	_arg_4200315755918524418_0 := "literal_arg"
	go func(_parentGid uint64) {
		goroutine.Enter(4200315755918524418, _parentGid)
		defer goroutine.Exit(4200315755918524418)
		println(_arg_4200315755918524418_0)
	}(goroutine.CurrentGid())

	local := 100
	_arg_4200315755918524419_0 := getValue() + local
	go func(_parentGid uint64) {
		goroutine.Enter(4200315755918524419, _parentGid)
		defer goroutine.Exit(4200315755918524419)
		foo(_arg_4200315755918524419_0)
	}(goroutine.CurrentGid())
	_arg_4200315755918524420_0 := len([]int{1, 2, 3})
	go func(_parentGid uint64) {
		goroutine.Enter(4200315755918524420, _parentGid)
		defer goroutine.Exit(4200315755918524420)
		foo(_arg_4200315755918524420_0)
	}(goroutine.CurrentGid())
}
func TestDeferEval_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()

	ch1 := make(chan int, 1)
	_ch_4200315755918524421 := ch1
	defer func() {
		sched.InstChBF(4200315755918524421)
		close(_ch_4200315755918524421)
		sched.InstChAF(4200315755918524421, _ch_4200315755918524421, "close")
	}()

	var wg sync.WaitGroup
	sched.InstWgBF(4200315755918524430)
	wg.Add(1)
	sched.InstWgAF(4200315755918524430, &wg, "add")
	_wg_4200315755918524431 := wg
	defer func() {
		sched.InstWgBF(4200315755918524431)
		_wg_4200315755918524431.Done()
		sched.InstWgAF(4200315755918524431, &_wg_4200315755918524431, "done")
	}()

	var wg2 sync.WaitGroup
	sched.InstWgBF(4200315755918524432)
	wg2.Add(1)
	sched.InstWgAF(4200315755918524432, &wg2, "add")
	_wg_4200315755918524433 := wg2
	defer func() {
		sched.InstWgBF(4200315755918524433)
		_wg_4200315755918524433.Add(-1)
		sched.InstWgAF(4200315755918524433, &_wg_4200315755918524433, "add")
	}()
}
func TestChannelEval_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	_ch_4200315755918524422 := getChan()
	sched.InstChBF(4200315755918524422)
	_ch_4200315755918524422 <- 42
	sched.InstChAF(4200315755918524422, _ch_4200315755918524422, "send")

	w := &Wrapper{Ch0: make(chan int, 1)}
	_ch_4200315755918524423 := w.Ch0
	sched.InstChBF(4200315755918524423)
	_ch_4200315755918524423 <- 100
	sched.InstChAF(4200315755918524423, _ch_4200315755918524423, "send")
	_ch_4200315755918524424 := w.getChan2()
	sched.InstChBF(4200315755918524424)
	_ch_4200315755918524424 <- 20
	sched.InstChAF(4200315755918524424, _ch_4200315755918524424, "send")
}
func TestCloseExpr_1(t *testing.T) {
	goroutine.EnterMain()
	defer goroutine.ExitMain()
	goroutine.ParseInput()
	sched.ParseInput()
	defer goroutine.PrintGoroutinePairs()
	_ch_4200315755918524425 := chMap["a"]
	sched.InstChBF(4200315755918524425)
	close(_ch_4200315755918524425)
	sched.InstChAF(4200315755918524425, _ch_4200315755918524425, "close")

	w := &Wrapper{Ch0: make(chan int, 1)}
	_ch_4200315755918524426 := w.Ch0
	sched.InstChBF(4200315755918524426)
	close(_ch_4200315755918524426)
	sched.InstChAF(4200315755918524426, _ch_4200315755918524426, "close")
	_ch_4200315755918524427 := getChan()
	sched.InstChBF(4200315755918524427)
	close(_ch_4200315755918524427)
	sched.InstChAF(4200315755918524427, _ch_4200315755918524427, "close")

	ch := make(chan int, 1)
	_ch_4200315755918524428 := ch
	sched.InstChBF(4200315755918524428)
	close(_ch_4200315755918524428)
	sched.InstChAF(4200315755918524428, _ch_4200315755918524428, "close")

	ch2 := make(chan int, 1)
	_ch_4200315755918524429 := ch2
	defer func() {
		sched.InstChBF(4200315755918524429)
		close(_ch_4200315755918524429)
		sched.InstChAF(4200315755918524429, _ch_4200315755918524429, "close")
	}()
}
