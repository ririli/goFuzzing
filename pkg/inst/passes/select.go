package passes

import (
	"go/ast"
	"io/ioutil"
	"log"
	"toolkit/pkg/inst"
	"toolkit/pkg/utils/gofmt"

	"golang.org/x/tools/go/ast/astutil"
)

var (
	SelectInstNeed = "SelectNeedInst"
	// 导入常量复用 global.go 的 OperationImportName/OperationImportPath
)

type SelectPass struct {
}

func RunSelectPass(in, out string) error {
	p := SelectPass{}
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

func (p *SelectPass) Before(iCtx *inst.InstContext) {
	iCtx.SetMetadata(SelectInstNeed, false)
}

func (p *SelectPass) After(iCtx *inst.InstContext) {
	need, _ := iCtx.GetMetadata(SelectInstNeed)
	needinst := need.(bool)
	if needinst {
		inst.AddImport(iCtx.FS, iCtx.AstFile, OperationImportName, OperationImportPath)
	}
}

func (p *SelectPass) GetPostApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		return true
	}
}

func (p *SelectPass) GetPreApply(iCtx *inst.InstContext) func(*astutil.Cursor) bool {
	return func(c *astutil.Cursor) bool {
		defer func() {
			if r := recover(); r != nil {
			}
		}()

		switch concrete := c.Node().(type) {

		case *ast.SelectStmt:
			cases := concrete.Body.List
			for _, x := range cases {
				if _, ok := x.(*ast.CommClause); !ok {
					return true
				}
				comm, _ := x.(*ast.CommClause)
				switch concrete := comm.Comm.(type) {
				case *ast.SendStmt: // send
					id := iCtx.GetNewOpId()
					Add(concrete.Pos(), id)
					ch := concrete.Chan
					newCall := GenInstCallWithType("InstChSelectAF", ch, id, "send")
					comm.Body = append([]ast.Stmt{newCall}, comm.Body...)
					iCtx.SetMetadata(SelectInstNeed, true)
				}
			}
		}

		return true
	}
}
