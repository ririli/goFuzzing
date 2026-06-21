package feedback

import "fmt"

// GortPairInfo 表示一对并发goroutine的信息（用于序列化/反序列化）
// 对标 SuspiciousPairInfo，但字段语义为goroutine ID而非函数ID
type GortPairInfo struct {
	Gid1       uint64           // 第一个goroutine的ID
	Gid2       uint64           // 第二个goroutine的ID
	CallLoc1   CallLocationInfo // 第一个go语句的位置
	CallLoc2   CallLocationInfo // 第二个go语句的位置
	Confidence float64          // 置信度 (0.0-1.0)
	SourceType string           // 来源类型: "observed", "inferred_parent_child", "inferred_shared_obj", "inferred_self_pair"
	IsObserved bool             // 是否为直接观测到的（Confidence == 1.0）
}

// InputGortPair 传递给测试二进制的goroutine对输入
// 对标 InputPair
type InputGortPair struct {
	TryPair     []*GortPairInfo
	RecordStack bool // true: 记录调用栈; false: 仅断点控制
}

// ToString 返回字符串表示，用于设置Input环境变量
// 格式: (gid1,gid2)(gid3,gid4)...
func (p *InputGortPair) ToString() string {
	if p == nil || len(p.TryPair) == 0 {
		return ""
	}

	var result string
	for _, pair := range p.TryPair {
		if pair == nil {
			continue
		}
		result += fmt.Sprintf("(%d,%d)", pair.Gid1, pair.Gid2)
	}
	return result
}

// String 返回字符串表示（与 callstack 包中的格式一致）
func (g *GortPairInfo) String() string {
	if g.IsObserved {
		return fmt.Sprintf("[COVERED] %d,%d|%s:%d,%s:%d|%.2f|%s;\n",
			g.Gid1, g.Gid2,
			g.CallLoc1.File, g.CallLoc1.Line,
			g.CallLoc2.File, g.CallLoc2.Line,
			g.Confidence,
			g.SourceType)
	}
	return fmt.Sprintf("[SUSPECT] %d,%d|%s:%d,%s:%d|%.2f|%s;\n",
		g.Gid1, g.Gid2,
		g.CallLoc1.File, g.CallLoc1.Line,
		g.CallLoc2.File, g.CallLoc2.Line,
		g.Confidence,
		g.SourceType)
}
