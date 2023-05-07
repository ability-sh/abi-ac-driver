package driver

import "bytes"

type Router struct {
	items map[string]string
}

func NewRouter(items map[string]string) *Router {
	return &Router{items: items}
}

func (R *Router) NewReflectExecutor(s interface{}) Executor {
	return NewReflectExecutorWithGetName(s, func(name string, b *bytes.Buffer) string {
		v, ok := R.items[name]
		if ok {
			return v
		}
		return GetName(name, b)
	})
}
