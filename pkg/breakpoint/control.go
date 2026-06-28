package breakpoint

import (
	"fmt"
	"time"
)

// PointControl 实现函数对之间的断点控制。
// 如果 funcId 需要等待其前驱函数完成，则阻塞直到前驱完成或超时。
func (c *Config) PointControl(funcId uint64) {
	if !c.HasActive() {
		return
	}
	if !c.IsActive(funcId) {
		return
	}

	if c.DoWait(funcId) {
		preIds := c.FindPrev(funcId)
		if preIds != nil {
			for _, preId := range preIds {
				waiter := c.GetWaiter(preId)
				select {
				case <-waiter:
					c.WaitMapDec(funcId)
					fmt.Printf("{COVERED} {%v, %v}\n", preId, funcId)
				case <-time.After(c.Timeout):
					fmt.Printf("{TIMEOUT} {%v, %v}\n", preId, funcId)
				}
			}
		}
	}
	c.CompleteOperation(funcId)
}
