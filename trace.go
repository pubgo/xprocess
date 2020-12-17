package xprocess

import (
	"expvar"
	"reflect"

	"github.com/pubgo/dix/dix_trace"
	"github.com/pubgo/xerror/xerror_util"
	"go.uber.org/atomic"
)

func init() {
	dix_trace.With(func(_ *dix_trace.TraceCtx) {
		expvar.Publish("xprocess", expvar.Func(func() interface{} {
			var _data = make(map[string]int32)
			data.Range(func(key, value interface{}) bool {
				_data[xerror_util.CallerWithFunc(key.(reflect.Value).Interface())] = value.(*atomic.Int32).Load()
				return true
			})
			return _data
		}))
	})
}
