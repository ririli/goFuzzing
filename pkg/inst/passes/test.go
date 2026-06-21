package passes

import (
	"go/ast"
	"strings"
	"toolkit/pkg/inst"

	"golang.org/x/tools/go/ast/astutil"
)

type TestPass struct {
	Pos string
}

var (
	TestNeedInst   = "NEED_TEST_INST"
	GortImportName = "goroutine"
	GortImportPath = "toolkit/pkg/goroutine"
)

func (p *TestPass) Before(ctx *inst.InstContext) {
	ctx.SetMetadata(TestNeedInst, false)
}

func (p *TestPass) After(ctx *inst.InstContext) {
	need, _ := ctx.GetMetadata(TestNeedInst)
	needinst := need.(bool)
	if needinst {
		inst.AddImport(ctx.FS, ctx.AstFile, GortImportName, GortImportPath)
	}
}

func (p *TestPass) GetPreApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		defer func() {
			if r := recover(); r != nil { // This is allowed. If we insert node into nodes not in slice, we will meet a panic
				// For example, we may identified a receive in select and wanted to insert a function call before it, then this function will panic
			}
		}()

		switch concrete := c.Node().(type) {
		case *ast.FuncDecl:
			name := concrete.Name.Name
			params := concrete.Type.Params.List

			if len(params) != 1 {
				return true
			}

			check_ok := false
			names := params[0].Names
			if len(names) != 1 || names[0].Name != "t" {
				return false
			}

			if v, ok := params[0].Type.(*ast.StarExpr); ok {
				if vv, ok := v.X.(*ast.SelectorExpr); ok {
					if vvv, ok := vv.X.(*ast.Ident); ok {
						if vv.Sel.Name == "T" && vvv.Name == "testing" {
							check_ok = true
						}
					}
				}
			}
			if check_ok && strings.HasPrefix(name, "Test") && !strings.HasSuffix(name, "_1") {
				testDecl := genTestDeclWithParseInput(name, concrete)
				iCtx.AstFile.Decls = append(iCtx.AstFile.Decls, testDecl)
				iCtx.SetMetadata(TestNeedInst, true)
			}
		}
		return true
	}
}

func (p *TestPass) GetPostApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return nil
}

func genTestDeclWithParseInput(name string, fn *ast.FuncDecl) *ast.FuncDecl {
	testname := name + "_1"

	// 创建 goroutine.EnterMain() 调用语句
	enterMainCall := &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "goroutine"},
				Sel: &ast.Ident{Name: "EnterMain"},
			},
			Args: []ast.Expr{},
		},
	}

	// 创建 defer goroutine.ExitMain() 调用语句
	exitMainDefer := &ast.DeferStmt{
		Call: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "goroutine"},
				Sel: &ast.Ident{Name: "ExitMain"},
			},
			Args: []ast.Expr{},
		},
	}

	// 创建 goroutine.ParseInput() 调用语句
	parseInputCall := &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "goroutine"},
				Sel: &ast.Ident{Name: "ParseInput"},
			},
			Args: []ast.Expr{},
		},
	}

	// 创建 sched.ParseInput() 调用语句（解析 InputOp 环境变量，使操作对调度生效）
	schedParseInputCall := &ast.ExprStmt{
		X: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "sched"},
				Sel: &ast.Ident{Name: "ParseInput"},
			},
			Args: []ast.Expr{},
		},
	}

	// 创建 defer goroutine.PrintGoroutinePairs() 调用语句
	printGoroutinePairsCall := &ast.DeferStmt{
		Call: &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "goroutine"},
				Sel: &ast.Ident{Name: "PrintGoroutinePairs"},
			},
			Args: []ast.Expr{},
		},
	}

	// 复制原始函数体语句
	testbodylst := make([]ast.Stmt, len(fn.Body.List))
	copy(testbodylst, fn.Body.List)

	// 在函数体开头插入 goroutine.EnterMain()、defer goroutine.ExitMain()、
	// goroutine.ParseInput()、sched.ParseInput() 和 defer goroutine.PrintGoroutinePairs()
	block := &ast.BlockStmt{
		List: append([]ast.Stmt{enterMainCall, exitMainDefer, parseInputCall, schedParseInputCall, printGoroutinePairsCall}, testbodylst...),
	}

	testdecl := &ast.FuncDecl{
		Name: &ast.Ident{Name: testname},
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{
						Names: []*ast.Ident{
							&ast.Ident{Name: "t"},
						},
						Type: &ast.StarExpr{
							X: &ast.SelectorExpr{
								X: &ast.Ident{
									Name: "testing",
								},
								Sel: &ast.Ident{
									Name: "T",
								},
							},
						},
					},
				},
			},
		},
		Body: block,
	}
	return testdecl
}
