package passes

import (
	"go/ast"
	"go/token"
	"strconv"
	"toolkit/pkg/inst"

	"golang.org/x/tools/go/ast/astutil"
)

// GoroutinePass 插桩go语句，为每个goroutine分配唯一ID并包装生命周期hook
// 将 go f(args) 转换为:
//
//	go func() {
//	    goroutine.Enter(id)
//	    defer goroutine.Exit(id)
//	    f(args)
//	}()
//
// 将 go func(){body}() 转换为:
//
//	go func() {
//	    goroutine.Enter(id)
//	    defer goroutine.Exit(id)
//	    func(){body}()
//	}()
var (
	GoroutineInstNeed   = "GoroutineNeedInst"
	GoroutineImportName = "goroutine"
	GoroutineImportPath = "toolkit/pkg/goroutine"
)

type GoroutinePass struct{}

func (p *GoroutinePass) Before(iCtx *inst.InstContext) {
	iCtx.SetMetadata(GoroutineInstNeed, false)
}

func (p *GoroutinePass) After(iCtx *inst.InstContext) {
	need, _ := iCtx.GetMetadata(GoroutineInstNeed)
	needinst := need.(bool)
	if needinst {
		inst.AddImport(iCtx.FS, iCtx.AstFile, GoroutineImportName, GoroutineImportPath)
	}
}

func (p *GoroutinePass) GetPreApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		defer func() {
			if r := recover(); r != nil {
				// This is allowed. If we insert node into nodes not in slice, we will meet a panic
			}
		}()

		switch concrete := c.Node().(type) {
		case *ast.GoStmt:
			id := iCtx.GetNewOpId()
			Add(concrete.Pos(), id) // 注册goroutine ID到全局map（供其他pass查找）

			newStmt := wrapGoStmt(concrete, id)
			c.Replace(newStmt)
			iCtx.SetMetadata(GoroutineInstNeed, true)
		}
		return true
	}
}

func (p *GoroutinePass) GetPostApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return nil
}

// wrapGoStmt 将go语句包装为带生命周期hook的匿名函数调用
// 统一处理 go f(args) 和 go func(){body}() 两种形式
func wrapGoStmt(goStmt *ast.GoStmt, id uint64) *ast.GoStmt {
	// goroutine.Enter(id)
	enterCall := NewArgCallExpr("goroutine", "Enter", []ast.Expr{
		&ast.BasicLit{
			ValuePos: 0,
			Kind:     token.INT,
			Value:    strconv.FormatUint(id, 10),
		},
	})

	// defer goroutine.Exit(id)
	exitDefer := &ast.DeferStmt{
		Call: NewArgCall("goroutine", "Exit", []ast.Expr{
			&ast.BasicLit{
				ValuePos: 0,
				Kind:     token.INT,
				Value:    strconv.FormatUint(id, 10),
			},
		}),
	}

	// 函数体: { goroutine.Enter(id); defer goroutine.Exit(id); <原始调用> }
	body := &ast.BlockStmt{
		List: []ast.Stmt{
			enterCall,
			exitDefer,
			&ast.ExprStmt{X: goStmt.Call}, // 将原始调用作为最后一条语句
		},
	}

	return &ast.GoStmt{
		Go: goStmt.Go,
		Call: &ast.CallExpr{
			Fun: &ast.FuncLit{
				Type: &ast.FuncType{Params: &ast.FieldList{List: nil}},
				Body: body,
			},
			Args: []ast.Expr{},
		},
	}
}
