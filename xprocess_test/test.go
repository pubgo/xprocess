package xprocess_test

import (
	"reflect"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xerror/xerror_util"
)

type Test = test

type test struct {
	fn     interface{}
	params [][]interface{}
}

func (t *test) In(args ...interface{}) {
	var params [][]interface{}
	if len(t.params) == 0 {
		for _, arg := range args {
			params = append(params, []interface{}{arg})
		}
	} else {
		for _, p := range t.params {
			for _, arg := range args {
				params = append(params, append(p, arg))
			}
		}
	}
	t.params = params
}

func (t *test) Do() {
	vfn := xerror_util.Func(t.fn)
	for i := range t.params {
		vfn(t.params[i]...)
	}
	return
}

func TestFuncWith(fn interface{}) *test {
	xerror.Assert(fn == nil, "[fn] should not be nil")
	xerror.AssertFn(reflect.TypeOf(fn).Kind() != reflect.Func, func() error {
		return xerror.Fmt("kind: %s, name: %s", reflect.TypeOf(fn).Kind(), reflect.TypeOf(fn))
	})
	return &test{fn: fn}
}
