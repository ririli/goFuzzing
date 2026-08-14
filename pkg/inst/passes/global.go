package passes

import (
	"go/token"
	"sync"
)

// function 运行时包的导入名/路径，FunctionPass 与 TestPass 共用
var (
	FuncImportName = "function"
	FuncImportPath = "toolkit/pkg/function"
)

// operation 运行时包（OP 钩子/InputOp 解析）的导入名/路径，
// ChRecPass/SelectPass/WgPass/TestPass 共用
var (
	OperationImportName = "operation"
	OperationImportPath = "toolkit/pkg/operation"
)

var id_map sync.Map

func Add(pos token.Pos, id uint64) {
	id_map.Store(pos, id)
}

func Find(pos token.Pos) (uint64, bool) {
	if v, ok := id_map.Load(pos); ok {
		if vv, ok2 := v.(uint64); ok2 {
			return vv, true
		}
	}
	return 0, false
}
