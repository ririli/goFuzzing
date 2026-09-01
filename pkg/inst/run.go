package inst

import (
	"go/ast"

	"golang.org/x/tools/go/ast/astutil"
)

// runPasses 使用给定的插桩上下文执行指定的 Pass 列表
func runPasses(iCtx *InstContext, passes []InstPass) error {
	for _, p := range passes {
		err := RunPass(p, iCtx)
		if err != nil {
			return err
		}
	}
	return nil
}

// Run 使用给定的 Pass 名称列表和插桩上下文执行各 Pass。
func Run(iCtx *InstContext, r *PassRegistry, passNames []string) error {
	var passes = make([]InstPass, 0, len(passNames))
	for _, passName := range passNames {
		pass, err := r.GetNewPassInstance(passName)
		if err != nil {
			return err
		}
		passes = append(passes, pass)
	}
	return runPasses(iCtx, passes)
}

func RunPass(p InstPass, iCtx *InstContext) error {
	p.Before(iCtx)
	iCtx.AstFile = astutil.Apply(iCtx.AstFile, p.GetPreApply(iCtx), p.GetPostApply(iCtx)).(*ast.File)
	p.After(iCtx)
	return nil
}
