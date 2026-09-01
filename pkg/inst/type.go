package inst

import (
	"go/ast"
	"go/token"
	"go/types"
	"sync/atomic"

	"golang.org/x/tools/go/ast/astutil"
)

// InstContext 包含对单个 Golang 源码文件进行插桩所需的全部信息。
type InstContext struct {
	File            string
	OriginalContent []byte
	FS              *token.FileSet
	AstFile         *ast.File
	Type            *types.Info
	Metadata        map[string]interface{} // 用户可设置随插桩上下文携带的自定义元数据
	opid            uint64
}

// TODO : 将 opid 改为全局唯一，用于跨文件操作
func (i *InstContext) GetNewOpId() uint64 {
	return atomic.AddUint64(&i.opid, 1)
}

func (i *InstContext) SetMetadata(key string, value interface{}) {
	i.Metadata[key] = value
}

func (i *InstContext) GetMetadata(key string) (interface{}, bool) {
	val, exist := i.Metadata[key]
	return val, exist
}

// InstPass 定义了用于对单个 Golang 源码文件插桩的 Pass 形态
type InstPass interface {
	// Deps 返回依赖的 Pass 列表
	// Deps() []string

	Before(iCtx *InstContext)

	GetPreApply(iCtx *InstContext) func(*astutil.Cursor) bool

	GetPostApply(iCtx *InstContext) func(*astutil.Cursor) bool

	After(iCtx *InstContext)
}

type InstPassConstructor func() InstPass

// PassRegistry 记录所有已注册的 Pass
type PassRegistry struct {
	// pass 名 => pass 构造器
	n2p map[string]InstPassConstructor
	// 注册顺序
	o []string
}
