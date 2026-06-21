package passes

import (
	"go/ast"
	"toolkit/pkg/inst"

	"golang.org/x/tools/go/ast/astutil"
)

// FunctionPass 现在仅负责分配funcID（供channel/wg/select等pass使用）
// 不再生成 defer callstack.Trace(funcID)() 调用
// goroutine级别的追踪由 GoroutinePass 在 go 语句处实现

type FunctionPass struct{}

func (p *FunctionPass) Before(iCtx *inst.InstContext) {
	// 仅分配funcID，不做任何代码插桩
}

func (p *FunctionPass) After(iCtx *inst.InstContext) {
	// 无需添加import（不再生成 callstack.Trace 调用）
}

func (p *FunctionPass) GetPreApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		defer func() {
			if r := recover(); r != nil { // This is allowed. If we insert node into nodes not in slice, we will meet a panic
			}
		}()

		switch concrete := c.Node().(type) {
		case *ast.FuncDecl:
			// 仅分配funcID，供channel/wg/select等pass使用
			if concrete.Body != nil {
				id := iCtx.GetNewOpId()
				Add(concrete.Pos(), id)
			}
		case *ast.FuncLit:
			// 仅分配funcID，供channel/wg/select等pass使用
			if concrete.Body != nil {
				id := iCtx.GetNewOpId()
				Add(concrete.Pos(), id)
			}
		}

		return true
	}
}

func (p *FunctionPass) GetPostApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return nil
}
