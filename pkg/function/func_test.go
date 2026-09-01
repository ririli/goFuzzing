package function

import (
	"reflect"
	"testing"
)

// resetTestTracker 重置包级 tracker，避免测试间状态污染。
func resetTestTracker() {
	tracker = NewFuncTracker()
	skipRecord = false
}

func TestCurrentFuncStack_EmptyByDefault(t *testing.T) {
	resetTestTracker()

	if fids := CurrentFuncStack(); len(fids) != 0 {
		t.Errorf("CurrentFuncStack() = %v, want empty", fids)
	}
}

func TestCurrentFuncStack_TracksNestedTrace(t *testing.T) {
	resetTestTracker()

	end1 := Trace(5)
	end2 := Trace(7)

	// 栈从外层到内层：[5, 7]
	if fids := CurrentFuncStack(); !reflect.DeepEqual(fids, []uint64{5, 7}) {
		t.Errorf("CurrentFuncStack() = %v, want [5 7]", fids)
	}

	end2()
	if fids := CurrentFuncStack(); !reflect.DeepEqual(fids, []uint64{5}) {
		t.Errorf("after inner exit, CurrentFuncStack() = %v, want [5]", fids)
	}

	end1()
	if fids := CurrentFuncStack(); len(fids) != 0 {
		t.Errorf("after all exits, CurrentFuncStack() = %v, want empty", fids)
	}
}

func TestCurrentFuncStack_ExcludesMainFuncID(t *testing.T) {
	resetTestTracker()

	// EnterMain 压栈的 FuncID=0 不参与归属
	EnterMain()
	if fids := CurrentFuncStack(); len(fids) != 0 {
		t.Errorf("CurrentFuncStack() after EnterMain = %v, want empty", fids)
	}

	end := Trace(9)
	if fids := CurrentFuncStack(); !reflect.DeepEqual(fids, []uint64{9}) {
		t.Errorf("CurrentFuncStack() = %v, want [9]", fids)
	}
	end()
	ExitMain()
}
