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
	err    func() error
	val    func() []reflect.Value
}

func (v *futureValue) String() string {
	xerror.Next().Panic(v.Err())
	return valueStr(v.getVal()...)
}

func (v *futureValue) Get() []reflect.Value { return v.getVal() }
func (v *futureValue) Err() error {
	_ = v.getVal()
	if v.err == nil {
		return nil
	}

	return v.err()
}

func (v *futureValue) getVal() []reflect.Value {
	v.done.Do(func() {
		if v.val == nil {
			return
		}

		v.values = v.val()
		v.val = nil
	})

	return v.values
}

func (v *futureValue) Value(fn interface{}) (gErr error) {
	defer xerror.Resp(func(err xerror_abc.XErr) {
		gErr = xerror.WrapF(err, "input:%s, func:%s", valueStr(v.getVal()...), reflect.TypeOf(fn))
	})

	val := v.getVal()
	xerror.Next().Panic(v.Err())

	xerror_util.FuncValue(fn)(val...)
	return
}

var _futureValue = sync.Pool{New: func() interface{} { return &futureValue{} }}

func futureValueGet() *futureValue { return _futureValue.Get().(*futureValue) }
func futureValuePut(val *futureValue) {
	val.values = val.values[:0]
	val.done = sync.Once{}
	_futureValue.Put(val)
}
