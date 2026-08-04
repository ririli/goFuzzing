package passes

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
	"toolkit/pkg/inst"

	"golang.org/x/tools/go/ast/astutil"
)

func NewArgCall(strPkg, strCallee string, vecExprArg []ast.Expr) *ast.CallExpr {
	newIdentPkg := &ast.Ident{
		NamePos: token.NoPos,
		Name:    strPkg,
		Obj:     nil,
	}
	newIdentCallee := &ast.Ident{
		NamePos: token.NoPos,
		Name:    strCallee,
		Obj:     nil,
	}
	var newCall *ast.CallExpr
	if strPkg != "" {
		fun := &ast.SelectorExpr{
			X:   newIdentPkg,
			Sel: newIdentCallee,
		}
		newCall = &ast.CallExpr{
			Fun:      fun,
			Lparen:   token.NoPos,
			Args:     vecExprArg,
			Ellipsis: token.NoPos,
			Rparen:   token.NoPos,
		}
	} else {
		newCall = &ast.CallExpr{
			Fun:      newIdentCallee,
			Lparen:   token.NoPos,
			Args:     vecExprArg,
			Ellipsis: token.NoPos,
			Rparen:   token.NoPos,
		}
	}
	return newCall
}

func NewArgCallExpr(strPkg, strCallee string, vecExprArg []ast.Expr) *ast.ExprStmt {
	newCall := NewArgCall(strPkg, strCallee, vecExprArg)
	newExpr := &ast.ExprStmt{X: newCall}
	return newExpr
}

func NewDeferExpr(strPkg, strCallee string, vecExprArg []ast.Expr) *ast.DeferStmt {
	newCall := NewArgCall(strPkg, strCallee, vecExprArg)
	newExpr := &ast.DeferStmt{Call: newCall}
	return newExpr
}

func getSelectorCallerType(iCtx *inst.InstContext, selExpr *ast.SelectorExpr) string {
	if callerIdent, ok := selExpr.X.(*ast.Ident); ok {
		if callerIdent.Obj != nil {
			if objStmt, ok := callerIdent.Obj.Decl.(*ast.AssignStmt); ok {
				if objIdent, ok := objStmt.Lhs[0].(*ast.Ident); ok {
					if to := iCtx.Type.Defs[objIdent]; to == nil || to.Type() == nil {
						return ""
					} else {
						return to.Type().String()
					}

				}
			}
		}
	}

	return ""
}

func SelectorCallerHasTypes(iCtx *inst.InstContext, selExpr *ast.SelectorExpr, trueIfUnknown bool, tys ...string) bool {
	t := getSelectorCallerType(iCtx, selExpr)
	if t == "" && trueIfUnknown {
		return true
	}
	for _, ty := range tys {
		if ty == t {
			return true
		}
	}

	return false
}

func getContextString(iCtx *inst.InstContext, c *astutil.Cursor) string {
	return iCtx.FS.Position(c.Node().Pos()).String()
}

func IsTestFunc(n ast.Node) bool {
	switch concrete := n.(type) {
	case *ast.FuncDecl:
		name := concrete.Name.Name
		params := concrete.Type.Params.List

		if !strings.HasPrefix(name, "Test") {
			return false
		}

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

		return check_ok
	default:
		return false
	}
}

func GenInstCall(f string, ch ast.Expr, id uint64) *ast.ExprStmt {
	return NewArgCallExpr("sched", f, []ast.Expr{&ast.BasicLit{
		ValuePos: 0,
		Kind:     token.INT,
		Value:    strconv.FormatUint(id, 10),
	}, ch,
	})
}

// GenInstCallBF 生成单参数的 BF 插桩调用（仅 opId）
// 生成: sched.InstXxxBF(opId)
func GenInstCallBF(f string, id uint64) *ast.ExprStmt {
	return NewArgCallExpr("sched", f, []ast.Expr{
		&ast.BasicLit{Kind: token.INT, Value: strconv.FormatUint(id, 10)},
	})
}

// GenInstCallWithType 生成带操作类型的插桩调用
// 生成: sched.InstXxxBF(opId, obj, "opType")
func GenInstCallWithType(f string, obj ast.Expr, id uint64, opType string) *ast.ExprStmt {
	return NewArgCallExpr("sched", f, []ast.Expr{
		&ast.BasicLit{
			ValuePos: 0,
			Kind:     token.INT,
			Value:    strconv.FormatUint(id, 10),
		},
		obj,
		&ast.BasicLit{ // opType
			ValuePos: 0,
			Kind:     token.STRING,
			Value:    `"` + opType + `"`,
		},
	})
}
