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

// FuncEdge 表示一次执行中观测到的函数调用者-被调用者边。
type FuncEdge struct {
	Caller uint64 `json:"caller"`
	Callee uint64 `json:"callee"`
	Count  uint64 `json:"count"`
}

// ConcurrencyPair 接口方法 —— SuspiciousPairInfo 实现。

func (s *SuspiciousPairInfo) ID1() uint64                     { return s.FuncID1 }
func (s *SuspiciousPairInfo) ID2() uint64                     { return s.FuncID2 }
func (s *SuspiciousPairInfo) CallLocation1() CallLocationInfo { return s.CallLoc1 }
func (s *SuspiciousPairInfo) CallLocation2() CallLocationInfo { return s.CallLoc2 }
func (s *SuspiciousPairInfo) GetConfidence() float64          { return s.Confidence }
func (s *SuspiciousPairInfo) SetConfidence(c float64)         { s.Confidence = c }
func (s *SuspiciousPairInfo) GetSource() string               { return s.SourceType }
func (s *SuspiciousPairInfo) SetSource(src string)            { s.SourceType = src }
func (s *SuspiciousPairInfo) GetObserved() bool               { return s.IsObserved }
func (s *SuspiciousPairInfo) SetObserved(o bool)              { s.IsObserved = o }

// PairKey 返回归一化的去重键（较小 FuncID 在前，包含调用位置）。
func (s *SuspiciousPairInfo) PairKey() string { return funcPairKey(s) }

// SignalKey 返回归一化的信号匹配键（较小 FuncID 在前，不含调用位置）。
func (s *SuspiciousPairInfo) SignalKey() string { return simpleFuncSignalKey(s.FuncID1, s.FuncID2) }

// ConcurrencyEdge 接口方法 —— FuncEdge 实现。

func (e *FuncEdge) Parent() uint64   { return e.Caller }
func (e *FuncEdge) Child() uint64    { return e.Callee }
func (e *FuncEdge) GetCount() uint64 { return e.Count }

// funcPairKey 生成函数并发对的唯一去重键（基于 FuncID 和 CallLoc）。
func funcPairKey(pair *SuspiciousPairInfo) string {
	var fid1, fid2 uint64
	var loc1, loc2 CallLocationInfo

	if pair.FuncID1 <= pair.FuncID2 {
		fid1, fid2 = pair.FuncID1, pair.FuncID2
		loc1, loc2 = pair.CallLoc1, pair.CallLoc2
	} else {
		fid1, fid2 = pair.FuncID2, pair.FuncID1
		loc1, loc2 = pair.CallLoc2, pair.CallLoc1
	}

	return fmt.Sprintf("%d-%d|%s:%d-%s:%d",
		fid1, fid2,
		loc1.File, loc1.Line,
		loc2.File, loc2.Line)
}

// simpleFuncSignalKey 生成函数对的信号匹配键（仅基于 FuncID，不含调用位置）。
func simpleFuncSignalKey(fid1, fid2 uint64) string {
	if fid1 <= fid2 {
		return fmt.Sprintf("%d-%d", fid1, fid2)
	}
	return fmt.Sprintf("%d-%d", fid2, fid1)
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
