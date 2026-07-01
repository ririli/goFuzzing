package passes

import (
	"go/ast"
	"io/ioutil"
	"log"
	"toolkit/pkg/inst"
	"toolkit/pkg/utils/gofmt"

	"golang.org/x/tools/go/ast/astutil"
)

// ChResPass, Channel Record Pass. This pass instrumented at
// following four channel related operations:
// send, recv, make, close

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

		// channel send operation
		case *ast.SendStmt:
			id := iCtx.GetNewOpId()
			Add(concrete.Pos(), id)
			ch := concrete.Chan
			before := GenInstCallWithType("InstChBF", ch, id, "send")
			c.InsertBefore(before)
			after := GenInstCallWithType("InstChAF", ch, id, "send")
			c.InsertAfter(after)

			iCtx.SetMetadata(ChannelNeedInst, true)

		case *ast.ExprStmt:
			if callExpr, ok := concrete.X.(*ast.CallExpr); ok {
				if funcIdent, ok := callExpr.Fun.(*ast.Ident); ok {
					// channel close operation
					if funcIdent.Name == "close" {
						id := iCtx.GetNewOpId()
						Add(concrete.Pos(), id)
						args := callExpr.Args
						if len(args) == 1 {
							if ch, ok := args[0].(*ast.Ident); ok {
								before := GenInstCallWithType("InstChBF", ch, id, "close")
								c.InsertBefore(before)

								after := GenInstCallWithType("InstChAF", ch, id, "close")
								c.InsertAfter(after)

								iCtx.SetMetadata(ChannelNeedInst, true)
							}
						}
					}
				}
			}
		case *ast.DeferStmt:
			callExpr := concrete.Call
			if funcIdent, ok := callExpr.Fun.(*ast.Ident); ok {
				// channel close operation
				if funcIdent.Name == "close" {
					id := iCtx.GetNewOpId()
					Add(concrete.Pos(), id)
					args := callExpr.Args
					if len(args) == 1 {
						if ch, ok := args[0].(*ast.Ident); ok {
							before := GenInstCallWithType("InstChBF", ch, id, "close")
							after := GenInstCallWithType("InstChAF", ch, id, "close")

							body := &ast.BlockStmt{List: []ast.Stmt{
								before,
								NewArgCallExpr("", "close", callExpr.Args),
								after,
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
			}
			return false
		}
		return true
	}
}
