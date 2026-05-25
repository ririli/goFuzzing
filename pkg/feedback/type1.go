package feedback

import "fmt"

// CallLocationInfo 表示函数调用位置信息
type CallLocationInfo struct {
	File string // 源文件路径
	Line int    // 行号
}

// SuspiciousPairInfo 表示可疑并发对的信息（用于序列化/反序列化）
type SuspiciousPairInfo struct {
	FuncID1    uint64           // 第一个函数的 ID
	FuncID2    uint64           // 第二个函数的 ID
	CallLoc1   CallLocationInfo // 第一个函数的调用位置
	CallLoc2   CallLocationInfo // 第二个函数的调用位置
	Confidence float64          // 置信度 (0.0-1.0)
	SourceType string           // 来源类型: "observed", "inferred_parent1", "inferred_child1", "inferred_parent2", "inferred_child2"
	IsObserved bool             // 是否为直接观测到的（Confidence == 1.0）
}
type InputPair struct {
	TryPair     []*SuspiciousPairInfo
	RecordStack bool // true: 记录调用栈; false: 仅断点控制
}

// ToString 返回字符串表示
func (p *InputPair) ToString() string {
	if p == nil || len(p.TryPair) == 0 {
		return ""
	}

	var result string
	for _, pair := range p.TryPair {
		if pair == nil {
			continue
		}
		// 格式化为 (FuncID1,FuncID2)
		result += fmt.Sprintf("(%d,%d)", pair.FuncID1, pair.FuncID2)
	}
	return result
}

// String 返回字符串表示（与 callstack 包中的格式一致）
func (s *SuspiciousPairInfo) String() string {
	if s.IsObserved {
		return fmt.Sprintf("[COVERED] %d,%d|%s:%d,%s:%d|%.2f|%s;\n",
			s.FuncID1, s.FuncID2,
			s.CallLoc1.File, s.CallLoc1.Line,
			s.CallLoc2.File, s.CallLoc2.Line,
			s.Confidence,
			s.SourceType)
	}
	return fmt.Sprintf("[SUSPECT] %d,%d|%s:%d,%s:%d|%.2f|%s;\n",
		s.FuncID1, s.FuncID2,
		s.CallLoc1.File, s.CallLoc1.Line,
		s.CallLoc2.File, s.CallLoc2.Line,
		s.Confidence,
		s.SourceType)
}
