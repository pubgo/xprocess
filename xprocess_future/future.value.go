package xprocess_future

import (
	"reflect"
	"sync"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xerror/xerror_abc"
	"github.com/pubgo/xerror/xerror_util"
	"github.com/pubgo/xprocess/xprocess_abc"
)

var _ xprocess_abc.FutureValue = (*futureValue)(nil)

type futureValue struct {
	done   sync.Once
	values []reflect.Value
	err    error
	valFn  func() []reflect.Value
}

func (v *futureValue) setErr(err error) *futureValue { v.err = err; return v }
func (v *futureValue) Raw() []reflect.Value          { return v.getVal() }
func (v *futureValue) String() string                { return valueStr(v.getVal()...) }

func (v *futureValue) Get() interface{} {
	val := v.getVal()
	if len(val) == 0 || !val[0].IsValid() {
		return nil
	}

	return val[0].Interface()
}

func (v *futureValue) Err() error {
	_ = v.getVal()
	return v.err
}

func (v *futureValue) getVal() []reflect.Value {
	v.done.Do(func() {
		if v.valFn != nil {
			v.values = v.valFn()
		}
	})
	return v.values
}

func (v *futureValue) Value(fn interface{}) (gErr error) {
	defer xerror.Resp(func(err xerror_abc.XErr) {
		gErr = err.WrapF("input:%s, func:%s", valueStr(v.getVal()...), reflect.TypeOf(fn))
	})

	xerror.Assert(fn == nil, "[fn] should not be nil")
	xerror.Panic(v.Err())

	xerror_util.FuncValue(fn)(v.getVal()...)
	return
}

func futureValueGet() *futureValue    { return &futureValue{} }
func futureValuePut(val *futureValue) { _ = val }

var _ xprocess_abc.Value = (*value)(nil)

type value struct {
	err error
	val interface{}
}

func (v *value) Err() error         { return v.err }
func (v *value) String() string     { return "" }
func (v *value) Value() interface{} { return v.val }
