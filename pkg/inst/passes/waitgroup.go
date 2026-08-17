package passes

import (
	"fmt"
	"go/ast"
	"go/token"
	"toolkit/pkg/inst"

	"golang.org/x/tools/go/ast/astutil"
)

// isWaitGroupReceiver 依据类型检查结果判断接收者是否为 sync.WaitGroup。
// 类型未知时返回 false（宁缺毋滥），避免将 http.Header.Add 等
// 同名方法误判为 WaitGroup.Add 而生成无法编译的代码。
func isWaitGroupReceiver(iCtx *inst.InstContext, x ast.Expr) bool {
	tv, ok := iCtx.Type.Types[x]
	if !ok || tv.Type == nil {
		return false
	}
	switch tv.Type.String() {
	case "sync.WaitGroup", "*sync.WaitGroup":
		return true
	}
	return false
}

// wgReceiverArg 生成 InstWgAF 的接收者实参：
// typeSrc 用于查类型（可能是原接收者表达式），value 是实际传递的表达式。
// 指针类型直接传递，值类型取地址（此时 value 必为可寻址表达式）
func wgReceiverArg(iCtx *inst.InstContext, typeSrc, value ast.Expr) ast.Expr {
	if tv, ok := iCtx.Type.Types[typeSrc]; ok && tv.Type != nil && tv.Type.String() == "*sync.WaitGroup" {
		return value
	}
	return &ast.UnaryExpr{Op: token.AND, X: value}
}

// WgPass，WaitGroup Record Pass（WaitGroup 记录 Pass）。该 Pass 对
// sync.WaitGroup 的 Add、Done 操作进行插桩

var (
	WgNeedInst = "WgNeedInst"
	// 导入常量复用 global.go 的 OperationImportName/OperationImportPath
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
		inst.AddImport(iCtx.FS, iCtx.AstFile, OperationImportName, OperationImportPath)
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

			if !isWaitGroupReceiver(iCtx, selectorExpr.X) {
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
			before := GenInstCallBF("InstWgBF", id)
			c.InsertBefore(before)
			after := GenInstCallWithType("InstWgAF", wgReceiverArg(iCtx, wg, wg), id, opType)
			c.InsertAfter(after)
			iCtx.SetMetadata(WgNeedInst, true)

		case *ast.DeferStmt:
			callExpr := concrete.Call
			selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			if !isWaitGroupReceiver(iCtx, selectorExpr.X) {
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
			after := GenInstCallWithType("InstWgAF", wgReceiverArg(iCtx, selectorExpr.X, tmpIdent), id, opType)

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
