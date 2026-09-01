package feedback

import "fmt"

// GortEdge 表示一次执行中观测到的静态goroutine父子边。
type GortEdge struct {
	ParentGid uint64 `json:"parent"`
	ChildGid  uint64 `json:"child"`
	Count     uint64 `json:"count"`
}

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
	TryPair []*GortPairInfo
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

// ConcurrencyPair 接口方法 —— GortPairInfo 实现。

func (g *GortPairInfo) ID1() uint64                     { return g.Gid1 }
func (g *GortPairInfo) ID2() uint64                     { return g.Gid2 }
func (g *GortPairInfo) CallLocation1() CallLocationInfo { return g.CallLoc1 }
func (g *GortPairInfo) CallLocation2() CallLocationInfo { return g.CallLoc2 }
func (g *GortPairInfo) GetConfidence() float64          { return g.Confidence }
func (g *GortPairInfo) SetConfidence(c float64)         { g.Confidence = c }
func (g *GortPairInfo) GetSource() string               { return g.SourceType }
func (g *GortPairInfo) SetSource(src string)            { g.SourceType = src }
func (g *GortPairInfo) GetObserved() bool               { return g.IsObserved }
func (g *GortPairInfo) SetObserved(o bool)              { g.IsObserved = o }

// PairKey 返回归一化的去重键（较小 GID 在前，包含调用位置）。
func (g *GortPairInfo) PairKey() string { return GortPairKey(g) }

// SignalKey 返回归一化的信号匹配键（较小 GID 在前，不含调用位置）。
func (g *GortPairInfo) SignalKey() string { return gortSignalKey(g.Gid1, g.Gid2) }

// ConcurrencyEdge 接口方法 —— GortEdge 实现。

func (e *GortEdge) Parent() uint64   { return e.ParentGid }
func (e *GortEdge) Child() uint64    { return e.ChildGid }
func (e *GortEdge) GetCount() uint64 { return e.Count }

// GortPairKey 生成 goroutine 并发对的唯一去重键（基于 Gid 和 CallLoc）。
func GortPairKey(pair *GortPairInfo) string {
	var gid1, gid2 uint64
	var loc1, loc2 CallLocationInfo

	if pair.Gid1 <= pair.Gid2 {
		gid1, gid2 = pair.Gid1, pair.Gid2
		loc1, loc2 = pair.CallLoc1, pair.CallLoc2
	} else {
		gid1, gid2 = pair.Gid2, pair.Gid1
		loc1, loc2 = pair.CallLoc2, pair.CallLoc1
	}

	return fmt.Sprintf("%d-%d|%s:%d-%s:%d",
		gid1, gid2,
		loc1.File, loc1.Line,
		loc2.File, loc2.Line)
}

// gortSignalKey 生成 goroutine 对的信号匹配键（仅基于 GID，不含 CallLoc）。
func gortSignalKey(gid1, gid2 uint64) string {
	if gid1 <= gid2 {
		return fmt.Sprintf("%d-%d", gid1, gid2)
	}
	return fmt.Sprintf("%d-%d", gid2, gid1)
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
