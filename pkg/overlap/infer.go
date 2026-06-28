package overlap

import (
	"fmt"

	"toolkit/pkg/calltree"
)

// SuspiciousConcurrentPair 表示一个可疑的并发对
type SuspiciousConcurrentPair struct {
	Node1  *calltree.FunctionCallNode
	Node2  *calltree.FunctionCallNode
	Reason Source
}

// Source 记录可疑并发对的来源信息
type Source struct {
	Pair       *ConPairFunc
	SourceType string  // "observed" 或 "inferred_*"
	Confidence float64 // 置信度 0.0-1.0
}

// InferSuspiciousPairs 根据观测到的并发对推断可疑的并发对
//
// 推断规则（邻接规则）：
//
//	假设观测到 Node1 ↔ Node2 并发，则推断：
//	 0. Node1 × Node2 （直接观测，最高置信度）
//	 1. Node1.Parent × Node2
//	 2. Node1.Children × Node2
//	 3. Node1 × Node2.Parent
//	 4. Node1 × Node2.Children
func InferSuspiciousPairs(observedPairs []*ConPairFunc) []*SuspiciousConcurrentPair {
	var suspects []*SuspiciousConcurrentPair

	for _, observed := range observedPairs {
		if observed == nil || observed.Node1 == nil || observed.Node2 == nil {
			continue
		}

		addSuspect := func(node1, node2 *calltree.FunctionCallNode, confidence float64, sourceType string) {
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

		// 规则1：Node1的父节点 × Node2
		if observed.Node1.Parent != nil {
			addSuspect(observed.Node1.Parent, observed.Node2, 0.7, "inferred_parent1")
		}

		// 规则2：Node1的子节点 × Node2
		for i, child := range observed.Node1.Children {
			if i == 0 {
				addSuspect(child, observed.Node2, 0.7, "inferred_child1")
			} else {
				addSuspect(child, observed.Node2, 0.5, "inferred_child1")
			}
		}

		// 规则3：Node1 × Node2的父节点
		if observed.Node2.Parent != nil {
			addSuspect(observed.Node1, observed.Node2.Parent, 0.7, "inferred_parent2")
		}

		// 规则4：Node1 × Node2的子节点
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

	if s.Reason.Confidence == 1.0 {
		return fmt.Sprintf("[COVERED] %d,%d|%s:%d,%s:%d|%.2f|%s;\n",
			s.Node1.FuncID, s.Node2.FuncID,
			s.Node1.CallLoc.File, s.Node1.CallLoc.Line,
			s.Node2.CallLoc.File, s.Node2.CallLoc.Line,
			s.Reason.Confidence,
			s.Reason.SourceType)
	}

	return fmt.Sprintf("[SUSPECT] %d,%d|%s:%d,%s:%d|%.2f|%s;\n",
		s.Node1.FuncID, s.Node2.FuncID,
		s.Node1.CallLoc.File, s.Node1.CallLoc.Line,
		s.Node2.CallLoc.File, s.Node2.CallLoc.Line,
		s.Reason.Confidence,
		s.Reason.SourceType)
}
