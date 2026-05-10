package passes

import (
	"go/ast"
	"toolkit/pkg/inst"

	"golang.org/x/tools/go/ast/astutil"
)

var (
	FunctionInstNeed   = "FunctionNeedInst" //用于记录是否已经插桩了
	FunctionImportName = "callstack"
	FunctionImportPath = "toolkit/pkg/callstack"
)

type FunctionPass struct{}

func runFunctionPass(in, out string) error {
	return nil
}
func (p *FunctionPass) Before(iCtx *inst.InstContext) {

	iCtx.SetMetadata(FunctionInstNeed, false)
}
func (p *FunctionPass) After(iCtx *inst.InstContext) {
	need, _ := iCtx.GetMetadata(FunctionInstNeed)
	needinst := need.(bool)
	if needinst {
		inst.AddImport(iCtx.FS, iCtx.AstFile, FunctionImportName, FunctionImportPath)
	}
}

func (p *FunctionPass) GetPreApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		defer func() {
			if r := recover(); r != nil { // This is allowed. If we insert node into nodes not in slice, we will meet a panic
				// For example, we may identified a receive in select and wanted to insert a function call before it, then this function will panic
			}
		}()

		switch concrete := c.Node().(type) {
		case *ast.FuncDecl:
			// 检测到函数声明，在函数体开始处插桩
			if concrete.Body != nil && len(concrete.Body.List) > 0 {
				id := iCtx.GetNewOpId()
				Add(concrete.Pos(), id)

				st := GenInstFunction(id)
				concrete.Body.List = append([]ast.Stmt{st}, concrete.Body.List...)

				iCtx.SetMetadata(FunctionInstNeed, true)
			}
		}

		return true
	}
}

func (p *FunctionPass) GetPostApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return nil
}
