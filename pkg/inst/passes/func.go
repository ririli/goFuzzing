package passes

import (
	"go/ast"
	"go/token"
	"strconv"
	"toolkit/pkg/inst"

	"golang.org/x/tools/go/ast/astutil"
)

// FunctionPass 在函数入口注入 function.PointControl 和 function.Trace，
// 用于函数颗粒度 fuzzing。同时为 channel/wg/select 等 pass 分配 funcID。
type FunctionPass struct{}

func (p *FunctionPass) Before(iCtx *inst.InstContext) {}

func (p *FunctionPass) After(iCtx *inst.InstContext) {
	inst.AddImport(iCtx.FS, iCtx.AstFile, FuncImportName, FuncImportPath)
}

func (p *FunctionPass) GetPreApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		defer func() {
			if r := recover(); r != nil {
			}
		}()

		switch concrete := c.Node().(type) {
		case *ast.FuncDecl:
			if concrete.Body != nil && len(concrete.Body.List) > 0 {
				p.injectAtBody(concrete.Body, iCtx)
			}
		case *ast.FuncLit:
			if concrete.Body != nil && len(concrete.Body.List) > 0 {
				p.injectAtBody(concrete.Body, iCtx)
			}
		}

		return true
	}
}

// injectAtBody 在函数体开头插入断点控制和调用树追踪语句。
func (p *FunctionPass) injectAtBody(body *ast.BlockStmt, iCtx *inst.InstContext) {
	id := iCtx.GetNewOpId()
	Add(body.Pos(), id)

	idLit := &ast.BasicLit{
		Kind:  token.INT,
		Value: strconv.FormatUint(id, 10),
	}

	// function.PointControl(id)
	pcStmt := &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "function"},
				Sel: &ast.Ident{Name: "PointControl"},
			},
			Args: []ast.Expr{idLit},
		},
	}

	// defer function.Trace(id)()
	traceDefer := &ast.DeferStmt{
		Call: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "function"},
				Sel: &ast.Ident{Name: "Trace"},
			},
			Args: []ast.Expr{idLit},
		},
	}

	body.List = append([]ast.Stmt{pcStmt, traceDefer}, body.List...)
}

func (p *FunctionPass) GetPostApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return nil
}
