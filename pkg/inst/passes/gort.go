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
//	go func(_parentGid uint64) {
//	    goroutine.Enter(id, _parentGid)
//	    defer goroutine.Exit(id)
//	    f(args)
//	}(goroutine.CurrentGid())
//
// 将 go func(){body}() 转换为:
//
//	go func(_parentGid uint64) {
//	    goroutine.Enter(id, _parentGid)
//	    defer goroutine.Exit(id)
//	    func(){body}()
//	}(goroutine.CurrentGid())
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
// 通过匿名函数参数将父goroutine的gid传入子goroutine（参数在父goroutine中求值）
func wrapGoStmt(goStmt *ast.GoStmt, id uint64) *ast.GoStmt {
	// _parentGid 形参
	parentGidIdent := &ast.Ident{Name: "_parentGid"}
	paramField := &ast.Field{
		Names: []*ast.Ident{parentGidIdent},
		Type:  &ast.Ident{Name: "uint64"},
	}

	// goroutine.Enter(id, _parentGid)
	enterCall := NewArgCallExpr("goroutine", "Enter", []ast.Expr{
		&ast.BasicLit{
			ValuePos: 0,
			Kind:     token.INT,
			Value:    strconv.FormatUint(id, 10),
		},
		&ast.Ident{Name: "_parentGid"},
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

	// 函数体: { goroutine.Enter(id, _parentGid); defer goroutine.Exit(id); <原始调用> }
	body := &ast.BlockStmt{
		List: []ast.Stmt{
			enterCall,
			exitDefer,
			&ast.ExprStmt{X: goStmt.Call}, // 将原始调用作为最后一条语句
		},
	}

	// goroutine.CurrentGid() 调用 —— 作为匿名函数的实参，在父goroutine中求值
	currentGidCall := NewArgCall("goroutine", "CurrentGid", []ast.Expr{})

	return &ast.GoStmt{
		Go: goStmt.Go,
		Call: &ast.CallExpr{
			Fun: &ast.FuncLit{
				Type: &ast.FuncType{Params: &ast.FieldList{List: []*ast.Field{paramField}}},
				Body: body,
			},
			Args: []ast.Expr{currentGidCall},
		},
	}
}
