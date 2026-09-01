package passes

import (
	"go/token"
	"sync"
)

// 运行时包的导入名/路径，与各插桩 pass 共用。
// 导入别名一律使用 gopie_ 前缀，避免与被测项目中已有的
// 包级标识符（如 beego core/utils 自带的 func function(...)）冲突。
var (
	FuncImportName = "gopie_function"
	FuncImportPath = "toolkit/pkg/function"
)

// operation 运行时包（OP 钩子/InputOp 解析）的导入名/路径，
// ChRecPass/SelectPass/WgPass/TestPass 共用
var (
	OperationImportName = "gopie_operation"
	OperationImportPath = "toolkit/pkg/operation"
)

// goroutine 运行时包的导入名/路径，GoroutinePass/TestPass 共用
var (
	GortImportName = "gopie_goroutine"
	GortImportPath = "toolkit/pkg/goroutine"
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
