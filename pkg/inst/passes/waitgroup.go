package passes

import (
	"fmt"
	"go/ast"
	"go/token"
	"toolkit/pkg/inst"

	"golang.org/x/tools/go/ast/astutil"
)

// WgPass，WaitGroup Record Pass（WaitGroup 记录 Pass）。该 Pass 对
// sync.WaitGroup 的 Add、Done 操作进行插桩

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
			p_wg := &ast.UnaryExpr{Op: token.AND, X: wg}
			before := GenInstCallBF("InstWgBF", id)
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

			// 提取 receiver 到临时变量（defer 注册时求值，避免闭包延迟求值）
			tmpIdent := &ast.Ident{Name: fmt.Sprintf("_wg_%d", id)}
			tmpAssign := &ast.AssignStmt{
				Tok: token.DEFINE,
				Lhs: []ast.Expr{tmpIdent},
				Rhs: []ast.Expr{selectorExpr.X},
			}
			c.InsertBefore(tmpAssign)

			// 重建方法调用：_wg_N.Done() 或 _wg_N.Add(args...)
			tmpSelector := &ast.SelectorExpr{
				X:   tmpIdent,
				Sel: selectorExpr.Sel,
			}
			tmpCall := &ast.CallExpr{
				Fun:  tmpSelector,
				Args: callExpr.Args,
			}

			before := GenInstCallBF("InstWgBF", id)
			after := GenInstCallWithType("InstWgAF", &ast.UnaryExpr{Op: token.AND, X: tmpIdent}, id, opType)

			body := &ast.BlockStmt{List: []ast.Stmt{
				before,
				&ast.ExprStmt{X: tmpCall},
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
