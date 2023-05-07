package driver

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/ability-sh/abi-lib/errors"
	"github.com/ability-sh/abi-lib/eval"
	"github.com/ability-sh/abi-micro/micro"
)

type routerItem struct {
	Match    func(name string) (string, bool)
	Executor Executor
}

type Router struct {
	items []*routerItem
}

func NewRouter() *Router {
	return &Router{}
}

func (R *Router) Add(match func(name string) (string, bool), executor Executor) *Router {
	R.items = append(R.items, &routerItem{Match: match, Executor: executor})
	return R
}

func (R *Router) Rewrite(pattern *regexp.Regexp, to string, executor Executor) *Router {
	R.items = append(R.items, &routerItem{Match: func(name string) (string, bool) {
		vs := pattern.FindStringSubmatch(name)
		n := len(vs)
		if n > 0 {
			dst := eval.ParseEval(to, func(key string) string {
				i, _ := strconv.Atoi(key)
				if i < n {
					return vs[i]
				}
				return key
			})
			return dst, true
		}
		return "", false
	}, Executor: executor})
	return R
}

func (R *Router) Use(pattern *regexp.Regexp, executor Executor) *Router {
	R.items = append(R.items, &routerItem{Match: func(name string) (string, bool) {
		if pattern.MatchString(name) {
			return name, true
		}
		return "", false
	}, Executor: executor})
	return R
}

func (R *Router) Alias(alias string, executor Executor) *Router {
	n := len(alias)
	R.items = append(R.items, &routerItem{Match: func(name string) (string, bool) {
		if strings.HasPrefix(name, alias) {
			return name[n:], true
		}
		return "", false
	}, Executor: executor})
	return R
}

func (R *Router) Exec(ctx micro.Context, name string, data interface{}) (interface{}, error) {
	for _, item := range R.items {
		dst, ok := item.Match(name)
		if ok {
			return item.Executor.Exec(ctx, dst, data)
		}
	}
	return nil, errors.Errorf(404, "not found")
}
