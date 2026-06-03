package feedback

import "fmt"

// OpKind 操作对象类型枚举
type OpKind string

const (
	OpKindChannel   OpKind = "channel"
	OpKindWaitGroup OpKind = "waitgroup"
)

func (k OpKind) String() string { return string(k) }

// OpType 操作类型枚举
type OpType string

const (
	OpTypeSend  OpType = "send"
	OpTypeClose OpType = "close"
	OpTypeAdd   OpType = "add"
	OpTypeDone  OpType = "done"
)

func (t OpType) String() string { return string(t) }

// DangerType 危险类型枚举
type DangerType string

const (
	DangerCloseBeforeSend  DangerType = "close-before-send"
	DangerCloseBeforeClose DangerType = "close-before-close"
	DangerDoneBeforeAdd    DangerType = "done-before-add"
)

func (d DangerType) String() string { return string(d) }

// OpInfo 表示单个操作的信息（从 sched 日志解析）
type OpInfo struct {
	OpId     uint64 // 编译期唯一操作 ID
	FuncId   uint64 // 所在函数 ID
	ObjAddr  uint64 // 运行时对象地址（channel 指针或 wg 指针）
	OpType   OpType // 操作类型
	ObjKind  OpKind // 操作对象类型
	IsSelect bool   // 是否来自 select 分支（select 中的操作无 BF，只能做 Op1/pre）
}

// OpPair 表示一对操作同一对象且可能触发 panic 的配对
type OpPair struct {
	Op1    *OpInfo    // 先执行的操作
	Op2    *OpInfo    // 后执行的操作
	Danger DangerType // 危险类型
}
type InputOpPair struct {
	TryPair []*OpPair
}

// MatchOpPair 判断两个操作是否构成危险对（操作同一对象 + 危险组合）
// 返回 nil 表示不构成危险
func MatchOpPair(a, b *OpInfo) *OpPair {
	if a.ObjKind != b.ObjKind || a.ObjAddr != b.ObjAddr {
		return nil
	}

	switch a.ObjKind {
	case OpKindChannel:
		return matchChannelPair(a, b)
	case OpKindWaitGroup:
		return matchWgPair(a, b)
	}
	return nil
}

func matchChannelPair(a, b *OpInfo) *OpPair {
	// a=close, b=send → close-before-send
	if a.OpType == OpTypeClose && b.OpType == OpTypeSend {
		return &OpPair{Op1: a, Op2: b, Danger: DangerCloseBeforeSend}
	}
	// a=close, b=close → close-before-close
	if a.OpType == OpTypeClose && b.OpType == OpTypeClose && a.OpId != b.OpId {
		return &OpPair{Op1: a, Op2: b, Danger: DangerCloseBeforeClose}
	}
	return nil
}

func matchWgPair(a, b *OpInfo) *OpPair {
	// a=done, b=add → waitgroup counter 变负
	if a.OpType == OpTypeDone && b.OpType == OpTypeAdd {
		return &OpPair{Op1: a, Op2: b, Danger: DangerDoneBeforeAdd}
	}
	return nil
}

func (o *OpInfo) String() string {
	return fmt.Sprintf("[%s] opId=%d funcId=%d obj=0x%x op=%s",
		o.ObjKind, o.OpId, o.FuncId, o.ObjAddr, o.OpType)
}

func (p *OpPair) String() string {
	return fmt.Sprintf("%s: %s → %s", p.Danger, p.Op1, p.Op2)
}

func (p InputOpPair) ToString() string {
	if len(p.TryPair) == 0 {
		return ""
	}

	var result string
	for _, pair := range p.TryPair {
		if pair == nil || pair.Op1 == nil || pair.Op2 == nil {
			continue
		}
		result += fmt.Sprintf("(%d,%d)", pair.Op1.OpId, pair.Op2.OpId)
	}
	return result
}
