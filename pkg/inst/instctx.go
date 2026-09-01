package inst

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/ioutil"
	"log"
	"strings"
	"toolkit/pkg/utils/hash"
)

// NewInstContext 根据给定的 Golang 源文件创建 InstContext
func NewInstContext(goSrcFile string) (*InstContext, error) {
	oldSource, err := ioutil.ReadFile(goSrcFile)
	if err != nil {
		return nil, err
	}

	fs := token.NewFileSet()
	astF, err := parser.ParseFile(fs, goSrcFile, oldSource, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	stripCommentsForInst(astF)
	conf := types.Config{
		Importer: importer.ForCompiler(fs, "source", nil),
		Error: func(err error) {
			log.Printf("'%s' type checker: %s", goSrcFile, err)
		},
	}
	info := &types.Info{
		Types: make(map[ast.Expr]types.TypeAndValue),
		Defs:  make(map[*ast.Ident]types.Object),
		Uses:  make(map[*ast.Ident]types.Object),
	}

	conf.Check(astF.Name.Name, fs, []*ast.File{astF}, info)

	return &InstContext{
		File:            goSrcFile,
		OriginalContent: oldSource,
		FS:              fs,
		Type:            info,
		AstFile:         astF,
		Metadata:        make(map[string]interface{}),
		opid:            hash.Hash64(goSrcFile), // 使用64位hash作为ID前缀，避免跨文件碰撞
	}, nil
}

// stripCommentsForInst 剥离普通注释，仅保留 //go: 编译指令（如 //go:build）。
// 插桩会在语句间插入新节点，残留注释的位置关联会被 go/printer 错误地
// 织入新节点输出流（如把参数行尾注释插进注入的 defer 调用括号内），
// 产出无法编译的代码并触发 format.Node internal error。
func stripCommentsForInst(astF *ast.File) {
	if len(astF.Comments) == 0 {
		return
	}
	kept := make([]*ast.CommentGroup, 0, len(astF.Comments))
	for _, cg := range astF.Comments {
		directives := make([]*ast.Comment, 0, len(cg.List))
		for _, c := range cg.List {
			if strings.HasPrefix(c.Text, "//go:") {
				directives = append(directives, c)
			}
		}
		if len(directives) > 0 {
			cg.List = directives
			kept = append(kept, cg)
		}
	}
	astF.Comments = kept
}
