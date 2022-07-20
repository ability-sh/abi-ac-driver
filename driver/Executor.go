package driver

import "github.com/ability-sh/abi-micro/micro"

type Executor interface {
	Exec(ctx micro.Context, name string, data interface{}) (interface{}, error)
}
