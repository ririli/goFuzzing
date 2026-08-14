package passes

import (
	"fmt"
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
				// 这是允许的。如果向非切片中的节点插入节点，会触发 panic
			}
		}()

		switch concrete := c.Node().(type) {
		case *ast.GoStmt:
			id := iCtx.GetNewOpId()
			Add(concrete.Pos(), id) // 注册goroutine ID到全局map（供其他pass查找）

			// 提取 go 语句的参数到临时变量（在父 goroutine 中求值）
			args := concrete.Call.Args
			var argIdents []ast.Expr
			if len(args) > 0 {
				for i, arg := range args {
					tmpIdent := &ast.Ident{Name: fmt.Sprintf("_arg_%d_%d", id, i)}
					tmpAssign := &ast.AssignStmt{
						Tok: token.DEFINE,
						Lhs: []ast.Expr{tmpIdent},
						Rhs: []ast.Expr{arg},
					}
					c.InsertBefore(tmpAssign)
					argIdents = append(argIdents, tmpIdent)
				}
			}

			newStmt := wrapGoStmt(concrete, id, argIdents)
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
// argIdents: 已在父 goroutine 求值的临时变量引用，用于替换原始参数
func wrapGoStmt(goStmt *ast.GoStmt, id uint64, argIdents []ast.Expr) *ast.GoStmt {
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

	// 构建内部调用：原始函数 + 临时变量参数
	innerCall := goStmt.Call
	if len(argIdents) > 0 {
		innerCall = &ast.CallExpr{
			Fun:  goStmt.Call.Fun,
			Args: argIdents,
		}
	}

	// 函数体: { goroutine.Enter(id, _parentGid); defer goroutine.Exit(id); <innerCall> }
	body := &ast.BlockStmt{
		List: []ast.Stmt{
			enterCall,
			exitDefer,
			&ast.ExprStmt{X: innerCall},
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
