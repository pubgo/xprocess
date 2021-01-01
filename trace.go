package xprocess

import (
	"reflect"

	"github.com/pubgo/dix/dix_trace"
	"github.com/pubgo/xerror/xerror_util"
	"go.uber.org/atomic"
)

func init() {
	dix_trace.With(func(ctx *dix_trace.Ctx) {
		ctx.Func("xprocess", func() interface{} {
			var _data = make(map[string]int32)
			data.Range(func(key, value interface{}) bool {
				_data[xerror_util.CallerWithFunc(key.(reflect.Value).Interface())] = value.(*atomic.Int32).Load()
				return true
			})
			return _data
		})
	})
}
