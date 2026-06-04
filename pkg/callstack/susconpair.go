package callstack

import "fmt"

// SuspiciousConcurrentPair 表示一个可疑的并发对
type SuspiciousConcurrentPair struct {
	Node1  *FunctionCallNode
	Node2  *FunctionCallNode
	Reason Source
}

type Source struct {
	Pair       *ConPairFunc
	SourceType string  // "observed" 或 "inferred"
	Confidence float64 // 置信度 0.0-1.0
}

// InferSuspiciousPairs 根据观测到的并发对推断可疑的并发对
// 输入：观测到的并发对切片（ConPairFunc）
// 输出：推断出的可疑并发对切片（SuspiciousConcurrentPair）
//
// 推断规则（邻接规则）：
// 假设观测到 Node1 ↔ Node2 并发，则推断：
//  0. Node1 × Node2 （直接观测，最高置信度）
//  1. Node1.Parent × Node2 （父节点×Node2）
//  2. Node1.Children × Node2 （子节点×Node2）
//  3. Node1 × Node2.Parent （Node1×父节点）
//  4. Node1 × Node2.Children （Node1×子节点）
func InferSuspiciousPairs(observedPairs []*ConPairFunc) []*SuspiciousConcurrentPair {
	var suspects []*SuspiciousConcurrentPair

	for _, observed := range observedPairs {
		if observed == nil || observed.Node1 == nil || observed.Node2 == nil {
			continue
		}

		// 辅助函数：添加可疑对
		addSuspect := func(node1, node2 *FunctionCallNode, confidence float64, sourceType string) {
			if node1 == nil || node2 == nil {
				return
			}

			suspects = append(suspects, &SuspiciousConcurrentPair{
				Node1: node1,
				Node2: node2,
				Reason: Source{
					Pair:       observed,
					SourceType: sourceType,
					Confidence: confidence,
				},
			})
		}

		// 规则0：直接观测到的并发对（最高置信度）
		addSuspect(observed.Node1, observed.Node2, 1.0, "observed")

		// 规则1：Node1的父节点 × Node2（调用栈中直接相邻，高置信度）
		if observed.Node1.Parent != nil {
			addSuspect(observed.Node1.Parent, observed.Node2, 0.7, "inferred_parent1")
		}

		// 规则2：Node1的子节点 × Node2（第一个子节点紧邻Node1，置信度更高）
		for i, child := range observed.Node1.Children {
			if i == 0 {
				addSuspect(child, observed.Node2, 0.7, "inferred_child1")
			} else {
				addSuspect(child, observed.Node2, 0.5, "inferred_child1")
			}
		}

		// 规则3：Node1 × Node2的父节点（调用栈中直接相邻，高置信度）
		if observed.Node2.Parent != nil {
			addSuspect(observed.Node1, observed.Node2.Parent, 0.7, "inferred_parent2")
		}

		// 规则4：Node1 × Node2的子节点（第一个子节点紧邻Node2，置信度更高）
		for i, child := range observed.Node2.Children {
			if i == 0 {
				addSuspect(observed.Node1, child, 0.7, "inferred_child2")
			} else {
				addSuspect(observed.Node1, child, 0.5, "inferred_child2")
			}
		}
	}

	return suspects
}

// String 返回可疑对的字符串表示
func (s SuspiciousConcurrentPair) String() string {
	if s.Node1 == nil || s.Node2 == nil {
		return "[SUSPECT] invalid\n"
	}

	// 置信度为 1.0 时，表示直接观测到的并发对，输出 [COVERED]
	if s.Reason.Confidence == 1.0 {
		return fmt.Sprintf("[COVERED] %d,%d|%s:%d,%s:%d|%.2f|%s;\n",
			s.Node1.FuncID, s.Node2.FuncID,
			s.Node1.CallLoc.File, s.Node1.CallLoc.Line,
			s.Node2.CallLoc.File, s.Node2.CallLoc.Line,
			s.Reason.Confidence,
			s.Reason.SourceType)
	}

	// 其他情况输出 [SUSPECT]
	return fmt.Sprintf("[SUSPECT] %d,%d|%s:%d,%s:%d|%.2f|%s;\n",
		s.Node1.FuncID, s.Node2.FuncID,
		s.Node1.CallLoc.File, s.Node1.CallLoc.Line,
		s.Node2.CallLoc.File, s.Node2.CallLoc.Line,
		s.Reason.Confidence,
		s.Reason.SourceType)
}
