package passes

import (
	"go/ast"
	"go/token"
	"strconv"
	"toolkit/pkg/inst"

	"golang.org/x/tools/go/ast/astutil"
)

// FunctionPass 在函数入口注入 gopie_function.PointControl 和 gopie_function.Trace，
// 用于函数颗粒度 fuzzing。同时为 channel/wg/select 等 pass 分配 funcID。
type FunctionPass struct{}

// FuncNeedInst 标记本文件是否实际注入过函数粒度钩子，
// 仅在注入过时才添加运行时包导入，避免 imported and not used
const FuncNeedInst = "FuncNeedInst"

func (p *FunctionPass) Before(iCtx *inst.InstContext) {
	iCtx.SetMetadata(FuncNeedInst, false)
}

func (p *FunctionPass) After(iCtx *inst.InstContext) {
	need, _ := iCtx.GetMetadata(FuncNeedInst)
	needinst := need.(bool)
	if needinst {
		inst.AddImport(iCtx.FS, iCtx.AstFile, FuncImportName, FuncImportPath)
	}
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

	// 以原第一条语句的位置作为插入语句的位置：无位置（NoPos）节点会让
	// go/printer 把函数体开头的注释锚点漂移，错位打印到 selector 中间
	// （defer gopie_function.\n// comment\nTrace(...)），生成代码不整洁且
	// 可能移动 //go: 指令类注释。见 .tmp/panic_analysis_safemap_20260818.md。
	pos := body.List[0].Pos()

	idLit := func() *ast.BasicLit {
		return &ast.BasicLit{
			ValuePos: pos,
			Kind:     token.INT,
			Value:    strconv.FormatUint(id, 10),
		}
	}

	// gopie_function.PointControl(id)
	pcStmt := &ast.ExprStmt{
		X: &ast.CallExpr{
			Lparen: pos,
			Fun: &ast.SelectorExpr{
				X:   &ast.Ident{NamePos: pos, Name: FuncImportName},
				Sel: &ast.Ident{NamePos: pos, Name: "PointControl"},
			},
			Args:   []ast.Expr{idLit()},
			Rparen: pos,
		},
	}

	// defer gopie_function.Trace(id)()
	// 外层必须是「对 Trace 返回值的调用」：Trace 在入口求值并返回闭包，
	// 闭包才在函数退出时回填 EndTime。若只写 defer Trace(id)，
	// Trace 本身会被推迟到退出时才调用，返回的闭包被丢弃，
	// 所有实例的 EndTime 恒为 0，detectOverlaps 产不出任何配对。
	traceDefer := &ast.DeferStmt{
		Defer: pos,
		Call: &ast.CallExpr{
			Lparen: pos,
			Fun: &ast.CallExpr{
				Lparen: pos,
				Fun: &ast.SelectorExpr{
					X:   &ast.Ident{NamePos: pos, Name: FuncImportName},
					Sel: &ast.Ident{NamePos: pos, Name: "Trace"},
				},
				Args:   []ast.Expr{idLit()},
				Rparen: pos,
			},
			Rparen: pos,
		},
	}

	body.List = append([]ast.Stmt{pcStmt, traceDefer}, body.List...)
	iCtx.SetMetadata(FuncNeedInst, true)
}

func (p *FunctionPass) GetPostApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return nil
}
