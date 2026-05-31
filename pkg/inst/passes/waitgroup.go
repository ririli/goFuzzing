package passes

import (
	"go/ast"
	"go/token"
	"toolkit/pkg/inst"

	"golang.org/x/tools/go/ast/astutil"
)

// WgPass, WaitGroup Record Pass. This pass instruments
// sync.WaitGroup operations: Add, Done, Wait

var (
	WgNeedInst   = "WgNeedInst"
	WgImportName = "sched"
	WgImportPath = "toolkit/pkg/sched"
)

type WgPass struct {
	funcIdStack []uint64
}

func (p *WgPass) currentFuncId() uint64 {
	if len(p.funcIdStack) > 0 {
		return p.funcIdStack[len(p.funcIdStack)-1]
	}
	return 0
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
		switch c.Node().(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			if len(p.funcIdStack) > 0 {
				p.funcIdStack = p.funcIdStack[:len(p.funcIdStack)-1]
			}
		}
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

		// 追踪函数边界，维护 funcId 栈
		case *ast.FuncDecl:
			if fid, ok := Find(concrete.Pos()); ok {
				p.funcIdStack = append(p.funcIdStack, fid)
			}
		case *ast.FuncLit:
			if fid, ok := Find(concrete.Pos()); ok {
				p.funcIdStack = append(p.funcIdStack, fid)
			}

		case *ast.ExprStmt:
			callExpr, ok := concrete.X.(*ast.CallExpr)
			if !ok {
				return true
			}
			selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			// 检查 receiver 类型是否为 sync.WaitGroup 或 *sync.WaitGroup
			if !SelectorCallerHasTypes(iCtx, selectorExpr, true, "sync.WaitGroup", "*sync.WaitGroup") {
				return true
			}

			var opType string
			switch selectorExpr.Sel.Name {
			case "Add":
				opType = "add"
			case "Done":
				opType = "done"
			case "Wait":
				opType = "wait"
			default:
				return true
			}

			id := iCtx.GetNewOpId()
			Add(concrete.Pos(), id)
			wg := selectorExpr.X
			// 取地址传给 InstWgBF/InstWgAF
			p_wg := &ast.UnaryExpr{
				Op: token.AND,
				X:  wg,
			}
			fid := p.currentFuncId()
			before := GenInstCallWithType("InstWgBF", p_wg, id, fid, opType)
			c.InsertBefore(before)
			after := GenInstCallWithType("InstWgAF", p_wg, id, fid, opType)
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
			case "Wait":
				opType = "wait"
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
			fid := p.currentFuncId()
			before := GenInstCallWithType("InstWgBF", p_wg, id, fid, opType)
			after := GenInstCallWithType("InstWgAF", p_wg, id, fid, opType)

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
