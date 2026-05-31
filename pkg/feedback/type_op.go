package feedback

import "fmt"

// OpInfo 表示单个操作的信息（从 sched 日志解析）
type OpInfo struct {
	OpId    uint64 // 编译期唯一操作 ID
	FuncId  uint64 // 所在函数 ID
	ObjAddr uint64 // 运行时对象地址（channel 指针或 wg 指针）
	OpType  string // "send" / "recv" / "close" / "add" / "done" / "wait"
	ObjKind string // "channel" / "waitgroup"
}

// OpPair 表示一对操作同一对象且可能触发 panic 的配对
type OpPair struct {
	Op1    *OpInfo // 先执行的操作
	Op2    *OpInfo // 后执行的操作
	Danger string  // 危险类型: "close-before-send" / "close-before-close" / "done-negative-count"
}

// MatchOpPair 判断两个操作是否构成危险对（操作同一对象 + 危险组合）
// 返回 nil 表示不构成危险
func MatchOpPair(a, b *OpInfo) *OpPair {
	if a.ObjKind != b.ObjKind || a.ObjAddr != b.ObjAddr {
		return nil
	}

	switch a.ObjKind {
	case "channel":
		return matchChannelPair(a, b)
	case "waitgroup":
		return matchWgPair(a, b)
	}
	return nil
}

func matchChannelPair(a, b *OpInfo) *OpPair {
	// a=close, b=send → close-before-send
	if a.OpType == "close" && b.OpType == "send" {
		return &OpPair{Op1: a, Op2: b, Danger: "close-before-send"}
	}
	// a=close, b=close → close-before-close
	if a.OpType == "close" && b.OpType == "close" && a.OpId != b.OpId {
		return &OpPair{Op1: a, Op2: b, Danger: "close-before-close"}
	}
	return nil
}

func matchWgPair(a, b *OpInfo) *OpPair {
	// a=done, b=add → waitgroup counter 变负
	if a.OpType == "done" && b.OpType == "add" {
		return &OpPair{Op1: a, Op2: b, Danger: "done-before-add"}
	}
	// a=wait, b=add → 死锁（wait 不等还没 add 的）
	if a.OpType == "wait" && b.OpType == "add" {
		return &OpPair{Op1: a, Op2: b, Danger: "wait-before-add"}
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
