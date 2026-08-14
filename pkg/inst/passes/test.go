package passes

import (
	"go/ast"
	"strings"
	"toolkit/pkg/inst"

	"golang.org/x/tools/go/ast/astutil"
)

type TestPass struct {
	Pos string
	// Granularity 取值 "goroutine" 或 "function"。
	// 构造方必须先经 cmd.NormalizeGranularity（或等价的 fuzzer.ParseGranularity）归一化，
	// 本 pass 内部仅对归一化后的值做精确比较。
	Granularity string
}

var (
	TestNeedInst    = "NEED_TEST_INST"
	GortImportName  = "goroutine"
	GortImportPath  = "toolkit/pkg/goroutine"
	SchedImportName = "sched"
	SchedImportPath = "toolkit/pkg/sched"
	// FuncImportName/FuncImportPath 定义在 global.go，与 FunctionPass 共用
)

func (p *TestPass) Before(ctx *inst.InstContext) {
	ctx.SetMetadata(TestNeedInst, false)
}

func (p *TestPass) After(ctx *inst.InstContext) {
	need, _ := ctx.GetMetadata(TestNeedInst)
	needinst := need.(bool)
	if !needinst {
		return
	}
	inst.AddImport(ctx.FS, ctx.AstFile, SchedImportName, SchedImportPath)
	if p.Granularity == "function" {
		inst.AddImport(ctx.FS, ctx.AstFile, FuncImportName, FuncImportPath)
	} else {
		inst.AddImport(ctx.FS, ctx.AstFile, GortImportName, GortImportPath)
	}
}

func (p *TestPass) GetPreApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		defer func() {
			if r := recover(); r != nil { // This is allowed. If we insert node into nodes not in slice, we will meet a panic
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
				testDecl := p.genTestDecl(name, concrete)
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

// genTestDecl 根据 Granularity 生成模式专用的 TestXxx_1 包装函数。
func (p *TestPass) genTestDecl(name string, fn *ast.FuncDecl) *ast.FuncDecl {
	testname := name + "_1"

	// 复制原始函数体语句
	testbodylst := make([]ast.Stmt, len(fn.Body.List))
	copy(testbodylst, fn.Body.List)

	var wrapperStmts []ast.Stmt

	if p.Granularity == "function" {
		// 函数模式：function + sched
		wrapperStmts = []ast.Stmt{
			// function.EnterMain()
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "function"},
						Sel: &ast.Ident{Name: "EnterMain"},
					},
				},
			},
			// defer function.ExitMain()
			&ast.DeferStmt{
				Call: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "function"},
						Sel: &ast.Ident{Name: "ExitMain"},
					},
				},
			},
			// function.ParseInput()
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "function"},
						Sel: &ast.Ident{Name: "ParseInput"},
					},
				},
			},
			// sched.ParseInput()
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "sched"},
						Sel: &ast.Ident{Name: "ParseInput"},
					},
				},
			},
			// defer function.PrintFunctionPairs()
			&ast.DeferStmt{
				Call: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "function"},
						Sel: &ast.Ident{Name: "PrintFunctionPairs"},
					},
				},
			},
		}
	} else {
		// goroutine 模式（默认）：goroutine + sched
		wrapperStmts = []ast.Stmt{
			// goroutine.EnterMain()
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "goroutine"},
						Sel: &ast.Ident{Name: "EnterMain"},
					},
				},
			},
			// defer goroutine.ExitMain()
			&ast.DeferStmt{
				Call: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "goroutine"},
						Sel: &ast.Ident{Name: "ExitMain"},
					},
				},
			},
			// goroutine.ParseInput()
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "goroutine"},
						Sel: &ast.Ident{Name: "ParseInput"},
					},
				},
			},
			// sched.ParseInput()
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "sched"},
						Sel: &ast.Ident{Name: "ParseInput"},
					},
				},
			},
			// defer goroutine.PrintGoroutinePairs()
			&ast.DeferStmt{
				Call: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: "goroutine"},
						Sel: &ast.Ident{Name: "PrintGoroutinePairs"},
					},
				},
			},
		}
	}

	block := &ast.BlockStmt{
		List: append(wrapperStmts, testbodylst...),
	}

	return &ast.FuncDecl{
		Name: &ast.Ident{Name: testname},
		Type: &ast.FuncType{
			Params: &ast.FieldList{
				List: []*ast.Field{
					&ast.Field{
						Names: []*ast.Ident{&ast.Ident{Name: "t"}},
						Type: &ast.StarExpr{
							X: &ast.SelectorExpr{
								X:   &ast.Ident{Name: "testing"},
								Sel: &ast.Ident{Name: "T"},
							},
						},
					},
				},
			},
		},
		Body: block,
	}
}
