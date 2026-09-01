package inst

func NewPassRegistry() *PassRegistry {
	return &PassRegistry{
		n2p: make(map[string]InstPassConstructor),
		o:   make([]string, 0),
	}
}

// Register 将唯一的一个 Pass 注册进注册表
func (r *PassRegistry) Register(name string, passc InstPassConstructor) error {
	_, exist := r.n2p[name]
	if exist {
		return &PassExistedError{Name: name}
	}
	r.n2p[name] = passc
	r.o = append(r.o, name)
	return nil
}

// GetNewPassInstance 返回指定名称的 Pass 的新实例
func (r *PassRegistry) GetNewPassInstance(name string) (InstPass, error) {
	c, exist := r.n2p[name]
	if exist {
		return c(), nil
	}

	return nil, &NoPassError{Name: name}
}

func (r *PassRegistry) ListOfPassNames() []string {
	passes := make([]string, 0, len(r.n2p))

	for _, n := range r.o {
		if r.HasPass(n) {
			passes = append(passes, n)
		}
	}
	return passes
}

// HasPass 若 Pass 已注册返回 true，否则返回 false
func (r *PassRegistry) HasPass(name string) bool {
	_, exist := r.n2p[name]
	return exist
}
