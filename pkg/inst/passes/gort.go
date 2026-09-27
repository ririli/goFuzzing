package passes

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strconv"
	"toolkit/pkg/inst"

	"golang.org/x/tools/go/ast/astutil"
)

// GoroutinePass 插桩go语句，为每个goroutine分配唯一ID并包装生命周期hook
// 将 go f(args) 转换为:
//
//	go func(_parentGid uint64) {
//	    gopie_goroutine.Enter(id, _parentGid)
//	    defer gopie_goroutine.Exit(id)
//	    f(args)
//	}(gopie_goroutine.CurrentGid())
//
// 将 go func(){body}() 转换为:
//
//	go func(_parentGid uint64) {
//	    gopie_goroutine.Enter(id, _parentGid)
//	    defer gopie_goroutine.Exit(id)
//	    func(){body}()
//	}(gopie_goroutine.CurrentGid())
//
// 对方法调用 go recvExpr.Method(args)，接收者表达式与实参一样
// 提前到父 goroutine 求值（_recv_N := recvExpr.Method），保持与原始 go 语句
// 一致的求值时机；否则接收者读会被搬进子 goroutine，破坏原有的
// 同步关系（如锁内读字段）并引入插桩独有的 data race。
var (
	GoroutineInstNeed = "GoroutineNeedInst"
	// 导入名/路径复用 global.go 的 GortImportName/GortImportPath
)

type GoroutinePass struct{}

func (p *GoroutinePass) Before(iCtx *inst.InstContext) {
	iCtx.SetMetadata(GoroutineInstNeed, false)
}

func (p *GoroutinePass) After(iCtx *inst.InstContext) {
	need, _ := iCtx.GetMetadata(GoroutineInstNeed)
	needinst := need.(bool)
	if needinst {
		inst.AddImport(iCtx.FS, iCtx.AstFile, GortImportName, GortImportPath)
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

			// 方法调用的接收者提前求值到父 goroutine（与实参同理）。
			// 原 go 语句的接收者在父 goroutine、go 语句处求值；若原样保留在
			// 包装闭包内，接收者读会搬到子 goroutine 执行，丢失原有同步上下文
			// （如锁保护），制造插桩独有的 race 与 nil 解引用风险。
			var recvIdent ast.Expr
			if sel, ok := concrete.Call.Fun.(*ast.SelectorExpr); ok && sel.X != nil && isRuntimeValueExpr(iCtx, sel.X) {
				tmp := &ast.Ident{Name: fmt.Sprintf("_recv_%d", id)}
				c.InsertBefore(&ast.AssignStmt{
					Tok: token.DEFINE,
					Lhs: []ast.Expr{tmp},
					Rhs: []ast.Expr{concrete.Call.Fun},
				})
				recvIdent = tmp
			}
			// Function variables and factory/index expressions are also evaluated
			// by the parent. Capture the callable, not a copy of a value receiver.
			if recvIdent == nil {
				capture := false
				switch fun := concrete.Call.Fun.(type) {
				case *ast.Ident:
					_, capture = iCtx.Type.Uses[fun].(*types.Var)
				case *ast.CallExpr, *ast.IndexExpr, *ast.ParenExpr:
					capture = true
				}
				if capture {
					tmp := &ast.Ident{Name: fmt.Sprintf("_recv_%d", id)}
					c.InsertBefore(&ast.AssignStmt{Tok: token.DEFINE, Lhs: []ast.Expr{tmp}, Rhs: []ast.Expr{concrete.Call.Fun}})
					recvIdent = tmp
				}
			}

			// 提取 go 语句的参数到临时变量（在父 goroutine 中求值）
			args := concrete.Call.Args
			var argIdents []ast.Expr
			if len(args) > 0 {
				for i, arg := range args {
					// Keep constants in their original argument context: := would
					// default untyped 1 to int and break calls expecting e.g. int64.
					if tv, ok := iCtx.Type.Types[arg]; ok && tv.Value != nil {
						argIdents = append(argIdents, arg)
						continue
					}
					// 裸 nil 无法参与短变量声明（_arg := nil 会报
					// use of untyped nil），且作为常量无需提前求值，直接内联
					if ident, ok := arg.(*ast.Ident); ok && ident.Name == "nil" {
						argIdents = append(argIdents, arg)
						continue
					}
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

			newStmt := wrapGoStmt(concrete, id, argIdents, recvIdent)
			c.Replace(newStmt)
			iCtx.SetMetadata(GoroutineInstNeed, true)
		}
		return true
	}
}

func (p *GoroutinePass) GetPostApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return nil
}

// isRuntimeValueExpr 判断表达式是否为运行时值（变量/字段/复杂表达式），
// 以区别于包名与类型名（pkg.Func、方法表达式 T.Method 的接收者部分无需提前求值）。
//   - 非标识符（字段访问/索引/调用/解引用等）必为运行时值；
//   - 标识符依赖类型信息：仅 *types.Var（变量/字段/接收者）才提前求值，
//     类型未知时保守保持原样（包名误提取会导致生成代码无法编译）。
func isRuntimeValueExpr(iCtx *inst.InstContext, x ast.Expr) bool {
	ident, ok := x.(*ast.Ident)
	if !ok {
		return true
	}
	if obj, ok := iCtx.Type.Uses[ident]; ok {
		_, isVar := obj.(*types.Var)
		return isVar
	}
	return false
}

// wrapGoStmt 将go语句包装为带生命周期hook的匿名函数调用
// 统一处理 go f(args) 和 go func(){body}() 两种形式
// argIdents: 已在父 goroutine 求值的临时变量引用，用于替换原始参数
// recvIdent: 已在父 goroutine 求值的可调用值（无需捕获时为 nil）
func wrapGoStmt(goStmt *ast.GoStmt, id uint64, argIdents []ast.Expr, recvIdent ast.Expr) *ast.GoStmt {
	// _parentGid 形参
	parentGidIdent := &ast.Ident{Name: "_parentGid"}
	paramField := &ast.Field{
		Names: []*ast.Ident{parentGidIdent},
		Type:  &ast.Ident{Name: "uint64"},
	}

	// gopie_goroutine.Enter(id, _parentGid)
	enterCall := NewArgCallExpr(GortImportName, "Enter", []ast.Expr{
		&ast.BasicLit{
			ValuePos: 0,
			Kind:     token.INT,
			Value:    strconv.FormatUint(id, 10),
		},
		&ast.Ident{Name: "_parentGid"},
	})

	// defer gopie_goroutine.Exit(id)
	exitDefer := &ast.DeferStmt{
		Call: NewArgCall(GortImportName, "Exit", []ast.Expr{
			&ast.BasicLit{
				ValuePos: 0,
				Kind:     token.INT,
				Value:    strconv.FormatUint(id, 10),
			},
		}),
	}

	// 构建内部调用：接收者/参数替换为已提前求值的临时变量
	// 保留 Ellipsis，避免 go f(args...) 丢失 ... 导致变参编译错误
	fun := goStmt.Call.Fun
	if recvIdent != nil {
		fun = recvIdent
	}
	args := goStmt.Call.Args
	if len(argIdents) > 0 {
		args = argIdents
	}
	innerCall := &ast.CallExpr{
		Fun:      fun,
		Args:     args,
		Ellipsis: goStmt.Call.Ellipsis,
	}

	// 函数体: { gopie_goroutine.Enter(id, _parentGid); defer gopie_goroutine.Exit(id); <innerCall> }
	body := &ast.BlockStmt{
		List: []ast.Stmt{
			enterCall,
			exitDefer,
			&ast.ExprStmt{X: innerCall},
		},
	}

	// gopie_goroutine.CurrentGid() 调用 —— 作为匿名函数的实参，在父goroutine中求值
	currentGidCall := NewArgCall(GortImportName, "CurrentGid", []ast.Expr{})

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
