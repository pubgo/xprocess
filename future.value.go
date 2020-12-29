package xprocess

import (
	"reflect"
	"sync"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xerror/xerror_util"
	"github.com/pubgo/xlog"
)

type Value interface {
	rawValues() []reflect.Value // block
	pipe(fn interface{}) Value  // async callback
	Value(fn interface{})       // async callback
	Get() interface{}           // block wait for value
}

var _ Value = (*futureValue)(nil)

type futureValue struct {
	val    func() []reflect.Value
	values []reflect.Value
	done   sync.Once
}

func (v *futureValue) pipe(fn interface{}) Value {
	var values = make(chan []reflect.Value)
	go func() {
		defer xerror.Resp(func(err xerror.XErr) { xlog.Error("futureValue.pipe panic", xlog.Any("err", err)) })
		values <- reflect.ValueOf(fn).Call(v.getVal())
	}()
	return &futureValue{val: func() []reflect.Value { return <-values }}
}

func (v *futureValue) getVal() []reflect.Value {
	v.done.Do(func() { v.values = v.val(); v.val = nil })
	return v.values
}

func (v *futureValue) rawValues() []reflect.Value {
	return v.getVal()
}

func (v *futureValue) Value(fn interface{}) {
	defer xerror.Resp(func(err xerror.XErr) { xerror.PanicF(err, xerror_util.CallerWithFunc(fn)) })
	reflect.ValueOf(fn).Call(v.getVal())
}

func (v *futureValue) Get() interface{} {
	values := v.getVal()
	if len(values) == 0 {
		return nil
	}

	if len(values) == 1 && values[0].IsValid() {
		return values[0].Interface()
	}

	if len(values) == 2 {
		if values[1].IsValid() && !values[1].IsNil() {
			xerror.Next().Panic(values[1].Interface().(error))
		}

		if !values[0].IsValid() || values[0].IsNil() {
			return nil
		}

		val := values[0].Interface()
		if val1, ok := val.(Value); ok {
			defer xerror.RespRaise("futureValue.Get")
			return val1.Get()
		}
		return val
	}

	return nil
}
