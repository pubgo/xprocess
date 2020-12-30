package xprocess

import (
	"reflect"
	"sync"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xerror/xerror_util"
	"github.com/pubgo/xlog"
)

type FutureValue interface {
	raw() []reflect.Value            // block
	Get() interface{}                // block wait for value
	pipe(fn interface{}) FutureValue // async callback
	Value(fn interface{})            // async callback
	Err(func(err error))             // async callback
}

var _ FutureValue = (*futureValue)(nil)

type futureValue struct {
	val     func() []reflect.Value
	values  []reflect.Value
	err     func() error
	done    sync.Once
	errCall func(err error)
}

func (v *futureValue) raw() []reflect.Value   { return v.getVal() }
func (v *futureValue) Err(fn func(err error)) { v.errCall = fn }
func (v *futureValue) checkErr(err error, fn interface{}) bool {
	if err == nil {
		return false
	}

	var fields = []xlog.Field{xlog.Any("err", err)}
	if fn != nil {
		fields = append(fields, xlog.String("func", xerror_util.CallerWithFunc(fn)))
	}
	if v.errCall == nil {
		xlog.Error("futureValue.checkErr", fields...)
		return true
	}

	v.errCall(err)
	return true
}

func (v *futureValue) pipe(fn interface{}) FutureValue {
	var values = make(chan []reflect.Value)
	var err error
	go func() {
		val := v.getVal()
		if err = v.err(); err != nil {
			v.checkErr(err, fn)
			values <- nil
			return
		}

		defer xerror.Resp(func(err1 xerror.XErr) { err = err1; values <- nil })
		values <- reflect.ValueOf(fn).Call(val)
	}()
	return &futureValue{val: func() []reflect.Value { return <-values }, err: func() error { return err }}
}

func (v *futureValue) getVal() []reflect.Value {
	v.done.Do(func() { v.values = v.val() })
	return v.values
}

func (v *futureValue) Value(fn interface{}) {
	go func() {
		val := v.getVal()
		if v.checkErr(v.err(), fn) {
			return
		}

		defer xerror.Resp(func(err xerror.XErr) { v.checkErr(v.err(), fn) })
		reflect.ValueOf(fn).Call(val)
	}()

}

func (v *futureValue) Get() interface{} {
	values := v.getVal()
	if v.checkErr(v.err(), nil) {
		return nil
	}

	if len(values) == 0 {
		return nil
	}

	if len(values) == 1 && values[0].IsValid() {
		return values[0].Interface()
	}

	if len(values) == 2 {
		if values[1].IsValid() && !values[1].IsNil() {
			v.checkErr(values[1].Interface().(error), nil)
			return nil
		}

		if !values[0].IsValid() || values[0].IsNil() {
			return nil
		}

		return values[0].Interface()
	}

	return nil
}
