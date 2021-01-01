package xprocess

import (
	"reflect"
	"sync"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xerror/xerror_util"
)

type FutureValue interface {
	Err() error
	String() string
	Get() []reflect.Value
	Value(fn interface{})
}

var _ FutureValue = (*futureValue)(nil)

type futureValue struct {
	val    func() []reflect.Value
	err    func() error
	values []reflect.Value
	done   sync.Once
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
	v.done.Do(func() { v.values = v.val() })
	return v.values
}

func (v *futureValue) Value(fn interface{}) {
	val := v.getVal()
	xerror.Next().Panic(v.Err())

	defer xerror.RespRaise(func(err xerror.XErr) error {
		return xerror.WrapF(err, "input:%s, func:%s", valueStr(val...), reflect.TypeOf(fn))
	})
	xerror_util.FuncValue(fn)(val...)
}

var _futureValue = sync.Pool{
	New: func() interface{} {
		return &futureValue{}
	},
}

func futureValueGet() *futureValue {
	return _futureValue.Get().(*futureValue)
}

func futureValuePut(val *futureValue) {
	val.values = val.values[:0]
	val.done = sync.Once{}
	_futureValue.Put(val)
}
