package inst

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"io/fs"
	"os"

	"golang.org/x/tools/go/ast/astutil"
)

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
