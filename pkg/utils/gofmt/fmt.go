package gofmt

import (
	"go/parser"
	"go/token"
)

// HasSyntaxError 检查文件是否为合法的 Go 源码。
// 用 go/parser 纯语法解析，不依赖 go 可执行文件：
// 旧实现调 `go fmt` 会实际改写文件、且在 go 不在 PATH 时误报。
func HasSyntaxError(goSrcFile string) bool {
	fset := token.NewFileSet()
	_, err := parser.ParseFile(fset, goSrcFile, nil, 0)
	return err != nil
}
