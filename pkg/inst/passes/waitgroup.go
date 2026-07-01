package passes

import (
	"go/ast"
	"go/token"
	"toolkit/pkg/inst"

	"golang.org/x/tools/go/ast/astutil"
)

// WgPass, WaitGroup Record Pass. This pass instruments
// sync.WaitGroup operations: Add, Done

var (
	WgNeedInst   = "WgNeedInst"
	WgImportName = "sched"
	WgImportPath = "toolkit/pkg/sched"
)

type WgPass struct {
}

func (p *WgPass) Before(iCtx *inst.InstContext) {
	iCtx.SetMetadata(WgNeedInst, false)
}

func (p *WgPass) After(iCtx *inst.InstContext) {
	need, _ := iCtx.GetMetadata(WgNeedInst)
	needinst := need.(bool)
	if needinst {
		inst.AddImport(iCtx.FS, iCtx.AstFile, WgImportName, WgImportPath)
	}
}

func (p *WgPass) GetPostApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		return true
	}
}

func (p *WgPass) GetPreApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		defer func() {
			if r := recover(); r != nil {
			}
		}()

		switch concrete := c.Node().(type) {

		case *ast.ExprStmt:
			callExpr, ok := concrete.X.(*ast.CallExpr)
			if !ok {
				return true
			}
			selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			if !SelectorCallerHasTypes(iCtx, selectorExpr, true, "sync.WaitGroup", "*sync.WaitGroup") {
				return true
			}

			var opType string
			switch selectorExpr.Sel.Name {
			case "Add":
				opType = "add"
			case "Done":
				opType = "done"
			default:
				return true
			}

			id := iCtx.GetNewOpId()
			Add(concrete.Pos(), id)
			wg := selectorExpr.X
			p_wg := &ast.UnaryExpr{
				Op: token.AND,
				X:  wg,
			}
			before := GenInstCallWithType("InstWgBF", p_wg, id, opType)
			c.InsertBefore(before)
			after := GenInstCallWithType("InstWgAF", p_wg, id, opType)
			c.InsertAfter(after)
			iCtx.SetMetadata(WgNeedInst, true)

		case *ast.DeferStmt:
			callExpr := concrete.Call
			selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			if !SelectorCallerHasTypes(iCtx, selectorExpr, true, "sync.WaitGroup", "*sync.WaitGroup") {
				return true
			}

			var opType string
			switch selectorExpr.Sel.Name {
			case "Add":
				opType = "add"
			case "Done":
				opType = "done"
			default:
				return true
			}

			id := iCtx.GetNewOpId()
			Add(concrete.Pos(), id)

			wg := selectorExpr.X
			p_wg := &ast.UnaryExpr{
				Op: token.AND,
				X:  wg,
			}
			before := GenInstCallWithType("InstWgBF", p_wg, id, opType)
			after := GenInstCallWithType("InstWgAF", p_wg, id, opType)

			body := &ast.BlockStmt{List: []ast.Stmt{
				before,
				&ast.ExprStmt{callExpr},
				after,
			}}

			deferStmt := &ast.DeferStmt{
				Call: &ast.CallExpr{
					Fun: &ast.FuncLit{
						Type: &ast.FuncType{Params: &ast.FieldList{List: nil}},
						Body: body,
					},
					Args: []ast.Expr{},
				},
			}
			c.Replace(deferStmt)
			iCtx.SetMetadata(WgNeedInst, true)
			return false
		}
		return true
	}
}
