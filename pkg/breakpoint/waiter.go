package breakpoint

// GetWaiter 获取或创建指定操作 ID 的等待 channel
func (c *Config) GetWaiter(id uint64) chan struct{} {
	if val, ok := c.Waiters.Load(id); ok {
		return val.(chan struct{})
	}

	newWaiter := make(chan struct{}, 1) // 缓冲为 1，避免发送时阻塞

	actual, _ := c.Waiters.LoadOrStore(id, newWaiter)
	return actual.(chan struct{})
}

// CompleteOperation 标记操作完成，通知所有等待者
func (c *Config) CompleteOperation(id uint64) {
	if val, ok := c.Waiters.LoadAndDelete(id); ok {
		ch := val.(chan struct{})
		select {
		case <-ch:
			// channel 已关闭，无需重复关闭
		default:
			close(ch)
		}
		return
	}
	// 向前引用：completeOperation 发生在任何 getWaiter 之前
	// 创建一个预先关闭的 channel，后续 getWaiter 会直接返回
	done := make(chan struct{})
	close(done)
	c.Waiters.LoadOrStore(id, done)
}
