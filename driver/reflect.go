package driver

import (
	"bytes"
	"log"
	"reflect"

	"github.com/ability-sh/abi-ac/ac"
	"github.com/ability-sh/abi-lib/dynamic"
	"github.com/ability-sh/abi-micro/micro"
)

type reflectExecutor struct {
	s    interface{}
	exec map[string]reflect.Value
}

var contextType = reflect.TypeOf((*micro.Context)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()

func getName(name string, b *bytes.Buffer) string {
	b.Reset()
	for i, r := range name {
		if r >= 'A' && r <= 'Z' {
			if i != 0 {
				b.WriteRune('/')
			}
			b.WriteRune(r + 32)
		} else {
			b.WriteRune(r)
		}
	}
	b.WriteString(".json")
	return b.String()
}

func NewReflectExecutor(s interface{}) Executor {
	rs := &reflectExecutor{s: s, exec: map[string]reflect.Value{}}

	v := reflect.ValueOf(s)
	t := v.Type()

	num := v.NumMethod()

	b := bytes.NewBuffer(nil)

	for i := 0; i < num; i++ {

		m := v.Method(i)

		inCount := m.Type().NumIn()

		if inCount != 2 {
			continue
		}

		outCount := m.Type().NumOut()

		if outCount != 2 {
			continue
		}

		if !m.Type().In(0).AssignableTo(contextType) {
			continue
		}

		inType := m.Type().In(1)

		if inType.Kind() != reflect.Ptr || inType.Elem().Kind() != reflect.Struct {
			continue
		}

		if !m.Type().Out(1).AssignableTo(errorType) {
			continue
		}

		name := t.Method(i).Name

		n := getName(name, b)

		rs.exec[n] = m

		log.Println(n, "=>", name)

	}

	return rs
}

func (r *reflectExecutor) Exec(ctx micro.Context, name string, data interface{}) (interface{}, error) {

	m, ok := r.exec[name]

	if ok {

		task := reflect.New(m.Type().In(1).Elem())

		dynamic.SetReflectValue(task, data)

		rs := m.Call([]reflect.Value{reflect.ValueOf(ctx), task})

		if rs != nil && len(rs) > 0 {

			if rs[1].CanInterface() && !rs[1].IsNil() {
				return nil, rs[1].Interface().(error)
			}

			if rs[0].CanInterface() {
				return rs[0].Interface(), nil
			}
		}

		return nil, ac.Errorf(404, "Not Return %s", name)

	} else {
		return nil, ac.Errorf(404, "Not Found %s", name)
	}
}
