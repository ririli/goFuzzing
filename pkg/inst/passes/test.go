package passes

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
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
	TestNeedInst = "NEED_TEST_INST"
	// GortImportName/GortImportPath 定义在 global.go，
	// FuncImportName/FuncImportPath、OperationImportName/OperationImportPath 同理
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
	inst.AddImport(ctx.FS, ctx.AstFile, OperationImportName, OperationImportPath)
	if p.Granularity == "function" {
		inst.AddImport(ctx.FS, ctx.AstFile, FuncImportName, FuncImportPath)
	} else {
		inst.AddImport(ctx.FS, ctx.AstFile, GortImportName, GortImportPath)
	}
}

func (p *TestPass) GetPreApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		defer func() {
			if r := recover(); r != nil { // 这是允许的。如果向非切片中的节点插入节点，会触发 panic
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
			if len(names) > 1 || concrete.Body == nil {
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
				testDecl := p.genTestDecl(iCtx, name, concrete)
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

// cloneBody 通过“序列化再重解析”深拷贝函数体语句。
// 若直接共享原函数的 AST 节点，同一批节点会被打印两次，
// go/printer 的位置信息会错乱，导致 format.Node internal error。
func cloneBody(fset *token.FileSet, body *ast.BlockStmt) []ast.Stmt {
	fallback := func() []ast.Stmt {
		cp := make([]ast.Stmt, len(body.List))
		copy(cp, body.List)
		return cp
	}
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, body); err != nil {
		return fallback()
	}
	src := "package p\nfunc _w() " + buf.String()
	nf, err := parser.ParseFile(token.NewFileSet(), "clone.go", src, 0)
	if err != nil || len(nf.Decls) == 0 {
		return fallback()
	}
	if fd, ok := nf.Decls[0].(*ast.FuncDecl); ok && fd.Body != nil {
		return fd.Body.List
	}
	return fallback()
}

// genTestDecl 根据 Granularity 生成模式专用的 TestXxx_1 包装函数。
func (p *TestPass) genTestDecl(iCtx *inst.InstContext, name string, fn *ast.FuncDecl) *ast.FuncDecl {
	testname := name + "_1"
	paramName := "_"
	if names := fn.Type.Params.List[0].Names; len(names) == 1 {
		paramName = names[0].Name
	}

	// 深拷贝原始函数体语句（避免与原函数共享 AST 节点）
	testbodylst := cloneBody(iCtx.FS, fn.Body)

	var wrapperStmts []ast.Stmt

	if p.Granularity == "function" {
		// 函数模式：function + operation
		wrapperStmts = []ast.Stmt{
			// gopie_function.EnterMain()
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: FuncImportName},
						Sel: &ast.Ident{Name: "EnterMain"},
					},
				},
			},
			// defer gopie_function.ExitMain()
			&ast.DeferStmt{
				Call: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: FuncImportName},
						Sel: &ast.Ident{Name: "ExitMain"},
					},
				},
			},
			// gopie_function.ParseInput()
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: FuncImportName},
						Sel: &ast.Ident{Name: "ParseInput"},
					},
				},
			},
			// gopie_operation.ParseInput()
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: OperationImportName},
						Sel: &ast.Ident{Name: "ParseInput"},
					},
				},
			},
			// defer gopie_function.PrintFunctionPairs()
			&ast.DeferStmt{
				Call: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: FuncImportName},
						Sel: &ast.Ident{Name: "PrintFunctionPairs"},
					},
				},
			},
		}
	} else {
		// goroutine 模式（默认）：goroutine + operation
		wrapperStmts = []ast.Stmt{
			// gopie_goroutine.EnterMain()
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: GortImportName},
						Sel: &ast.Ident{Name: "EnterMain"},
					},
				},
			},
			// defer gopie_goroutine.ExitMain()
			&ast.DeferStmt{
				Call: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: GortImportName},
						Sel: &ast.Ident{Name: "ExitMain"},
					},
				},
			},
			// gopie_goroutine.ParseInput()
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: GortImportName},
						Sel: &ast.Ident{Name: "ParseInput"},
					},
				},
			},
			// gopie_operation.ParseInput()
			&ast.ExprStmt{
				X: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: OperationImportName},
						Sel: &ast.Ident{Name: "ParseInput"},
					},
				},
			},
			// defer gopie_goroutine.PrintGoroutinePairs()
			&ast.DeferStmt{
				Call: &ast.CallExpr{
					Fun: &ast.SelectorExpr{
						X:   &ast.Ident{Name: GortImportName},
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
						Names: []*ast.Ident{&ast.Ident{Name: paramName}},
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
