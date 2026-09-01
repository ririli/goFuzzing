package inst

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"io/fs"
	"os"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

// keepDirectiveComments 仅保留 //go: 编译指令注释（如 //go:build），
// 其余注释全部丢弃，避免打印器把残留注释织入注入语句。
func keepDirectiveComments(comments []*ast.CommentGroup) []*ast.CommentGroup {
	if len(comments) == 0 {
		return nil
	}
	kept := make([]*ast.CommentGroup, 0, len(comments))
	for _, cg := range comments {
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
	return kept
}

func AddImport(fs *token.FileSet, ast *ast.File, name string, path string) error {
	for _, vecImportSpec := range astutil.Imports(fs, ast) {
		for _, importSpec := range vecImportSpec {
			if importSpec != nil && importSpec.Name != nil {

				if importSpec.Name.Name == name && importSpec.Path.Value == path { // 之前已插桩过
					// 惯用做法
					return nil
				}
			}

		}
	}

	ok := astutil.AddNamedImport(fs, ast, name, path)
	if !ok {
		return fmt.Errorf("failed to add import %s %s", name, path)
	}
	return nil
}

// DumpAstFile 将 AST 序列化到指定文件。
// 先写入同目录临时文件再原子替换，避免写入失败时
// 留下被截断/半截的目标文件（曾导致源文件变 0 字节）。
func DumpAstFile(fset *token.FileSet, astFile *ast.File, dstFile string) error {
	if astFile == nil {
		return fmt.Errorf("found nil ast file for %s", dstFile)
	}
	// 剥离原始注释：注入节点的 Position 为 NoPos，打印器按位置安放注释时
	// 会把原代码注释插进注入语句中间（如 defer xxx.Trace(...) 被注释截断）。
	// 打印器仅依据 File.Comments 列表输出注释；插桩输出不需要保留注释，
	// 但必须保留 //go: 编译指令（如 //go:build，否则带 build tag 的
	// 同名文件会同时参与编译，造成重复声明）。
	astFile.Comments = keepDirectiveComments(astFile.Comments)
	fi, err := os.Stat(dstFile)
	var mode fs.FileMode
	if err != nil {
		// 除“文件不存在”外，返回任何错误
		if !os.IsNotExist(err) {
			return err
		} else {
			// 如果文件不存在，使用默认权限
			mode = 0666
		}
	} else {
		// 如果文件存在，沿用原权限
		mode = fi.Mode()
	}

	tmpFile := dstFile + ".instmp"
	w, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	err = format.Node(w, fset, astFile)
	if cerr := w.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmpFile)
		return err
	}
	return os.Rename(tmpFile, dstFile)
}
