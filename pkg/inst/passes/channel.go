package passes

import (
	"fmt"
	"go/ast"
	"go/token"
	"io/ioutil"
	"log"
	"toolkit/pkg/inst"
	"toolkit/pkg/utils/gofmt"

	"golang.org/x/tools/go/ast/astutil"
)

// ChResPass，Channel Record Pass（通道记录 Pass）。该 Pass 对以下
// 四种通道相关操作进行插桩：
// send、recv、make、close

var (
	ChannelNeedInst   = "ChannelNeedInst"
	ChannelImportName = "sched"
	ChannelImportPath = "toolkit/pkg/sched"
)

type ChRecPass struct {
}

func RunChannelPass(in, out string) error {
	p := ChRecPass{}
	iCtx, err := inst.NewInstContext(in)
	if err != nil {
		log.Fatalf("Analysis source code failed %v", err)
	}
	p.Before(iCtx)
	iCtx.AstFile = astutil.Apply(iCtx.AstFile, p.GetPreApply(iCtx), p.GetPostApply(iCtx)).(*ast.File)
	p.After(iCtx)
	inst.DumpAstFile(iCtx.FS, iCtx.AstFile, out)
	if gofmt.HasSyntaxError(out) {
		err = ioutil.WriteFile(out, iCtx.OriginalContent, 0777)
		if err != nil {
			log.Panicf("failed to recover file '%s'", out)
		}
	}
	return nil
}

func (p *ChRecPass) Before(iCtx *inst.InstContext) {
	iCtx.SetMetadata(ChannelNeedInst, false)
}

func (p *ChRecPass) After(iCtx *inst.InstContext) {
	need, _ := iCtx.GetMetadata(ChannelNeedInst)
	needinst := need.(bool)
	if needinst {
		inst.AddImport(iCtx.FS, iCtx.AstFile, ChannelImportName, ChannelImportPath)
	}
}

func (p *ChRecPass) GetPostApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		return true
	}
}

func (p *ChRecPass) GetPreApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		defer func() {
			if r := recover(); r != nil {
			}
		}()

		switch concrete := c.Node().(type) {

		// channel send（发送）操作
		case *ast.SendStmt:
			id := iCtx.GetNewOpId()
			Add(concrete.Pos(), id)

			// 引入临时变量，避免 channel 表达式被双重求值
			tmpIdent := &ast.Ident{Name: fmt.Sprintf("_ch_%d", id)}
			tmpAssign := &ast.AssignStmt{
				Tok: token.DEFINE,
				Lhs: []ast.Expr{tmpIdent},
				Rhs: []ast.Expr{concrete.Chan},
			}
			c.InsertBefore(tmpAssign)

			// SendStmt 和 AF 都使用临时变量，channel 表达式只求值一次
			concrete.Chan = tmpIdent
			before := GenInstCallBF("InstChBF", id)
			c.InsertBefore(before)
			after := GenInstCallWithType("InstChAF", tmpIdent, id, "send")
			c.InsertAfter(after)

			iCtx.SetMetadata(ChannelNeedInst, true)

		case *ast.ExprStmt:
			if callExpr, ok := concrete.X.(*ast.CallExpr); ok {
				if funcIdent, ok := callExpr.Fun.(*ast.Ident); ok {
					// channel close（关闭）操作
					if funcIdent.Name == "close" {
						id := iCtx.GetNewOpId()
						Add(concrete.Pos(), id)
						args := callExpr.Args
						if len(args) == 1 {
							// 引入临时变量，不再限制 channel 表达式类型
							tmpIdent := &ast.Ident{Name: fmt.Sprintf("_ch_%d", id)}
							tmpAssign := &ast.AssignStmt{
								Tok: token.DEFINE,
								Lhs: []ast.Expr{tmpIdent},
								Rhs: []ast.Expr{args[0]},
							}
							c.InsertBefore(tmpAssign)

							// close 参数和 AF 都使用临时变量
							callExpr.Args[0] = tmpIdent
							before := GenInstCallBF("InstChBF", id)
							c.InsertBefore(before)
							after := GenInstCallWithType("InstChAF", tmpIdent, id, "close")
							c.InsertAfter(after)

							iCtx.SetMetadata(ChannelNeedInst, true)
						}
					}
				}
			}
		case *ast.DeferStmt:
			callExpr := concrete.Call
			if funcIdent, ok := callExpr.Fun.(*ast.Ident); ok {
				// channel close（关闭）操作
				if funcIdent.Name == "close" {
					id := iCtx.GetNewOpId()
					Add(concrete.Pos(), id)
					args := callExpr.Args
					if len(args) == 1 {
						// 临时变量在 defer 外部求值，保证求值时机与原 defer 参数语义一致
						tmpIdent := &ast.Ident{Name: fmt.Sprintf("_ch_%d", id)}
						tmpAssign := &ast.AssignStmt{
							Tok: token.DEFINE,
							Lhs: []ast.Expr{tmpIdent},
							Rhs: []ast.Expr{args[0]},
						}
						c.InsertBefore(tmpAssign)

						body := &ast.BlockStmt{List: []ast.Stmt{
							GenInstCallBF("InstChBF", id),
							NewArgCallExpr("", "close", []ast.Expr{tmpIdent}),
							GenInstCallWithType("InstChAF", tmpIdent, id, "close"),
						}}

						deferStmt := &ast.DeferStmt{
							Call: &ast.CallExpr{
								Fun: &ast.FuncLit{
									Type: &ast.FuncType{Params: &ast.FieldList{List: nil}},
									Body: body,
								},
								Args: []ast.Expr{},
							},
						}

						c.Replace(deferStmt)
						iCtx.SetMetadata(ChannelNeedInst, true)
					}
				}
			}
			return false
		}
		return true
	}
}
