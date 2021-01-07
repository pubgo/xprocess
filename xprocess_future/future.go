package xprocess_future

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xerror/xerror_util"
	"github.com/pubgo/xprocess/xprocess_abc"
	"github.com/pubgo/xprocess/xprocess_waitgroup"
	"go.uber.org/atomic"
)

type promise struct {
	wg        xprocess_waitgroup.WaitGroup
	data      chan xprocess_abc.FutureValue
	done      sync.Once
	cancelled atomic.Bool
	err       atomic.Error
}

func (s *promise) Await(val xprocess_abc.FutureValue, fn interface{}) {
	if s.cancelled.Load() {
		return
	}

	xerror.Next().Panic(val.Err())

	val.Value(fn)
}

func (s *promise) waitForClose()   { s.done.Do(func() { go func() { s.wg.Wait(); close(s.data) }() }) }
func (s *promise) Wait() error     { s.wg.Wait(); return s.err.Load() }
func (s *promise) Cancelled() bool { return s.cancelled.Load() }
func (s *promise) Cancel()         { s.cancelled.Store(true) }
func (s *promise) Yield(data interface{}, fn ...interface{}) {
	if s.cancelled.Load() {
		return
	}

	if val, ok := data.(xprocess_abc.FutureValue); ok {
		s.wg.Inc()
		if len(fn) > 0 {
			val = Await(val, fn[0])
		}

		go func() { s.data <- val }()
		return
	}

	if val, ok := data.(func()); ok {
		s.wg.Inc()
		go func() {
			defer s.wg.Done()
			defer xerror.Resp(func(err xerror.XErr) {
				s.err.Store(err)
				s.Cancel()
			})

			val()
		}()
		return
	}

	s.wg.Inc()
	value := futureValueGet()
	value.val = func() []reflect.Value { return []reflect.Value{reflect.ValueOf(data)} }
	go func() { s.data <- value }()
}

func (s *promise) Value(fn interface{}) {
	s.waitForClose()

	vfn := xerror_util.FuncValue(fn)
	for data := range s.data {
		func() {
			defer futureValuePut(data.(*futureValue))
			defer s.wg.Done()
			defer xerror.RespRaise(func(err xerror.XErr) error {
				s.Cancel()
				return xerror.WrapF(err, xerror_util.CallerWithFunc(fn))
			})
			vfn(data.Get()...)
		}()
	}
}

func Map(data interface{}, fn interface{}) interface{} {
	vfn := reflect.ValueOf(fn)
	vd := reflect.ValueOf(data)
	l := vd.Len()
	var values []xprocess_abc.FutureValue
	for i := 0; i < l; i++ {
		values = append(values, Async(vfn, vd.Index(i)))
	}

	var t reflect.Type
	if vfn.Type().NumOut() > 0 {
		t = vfn.Type().Out(0)
	}

	xerror.Assert(t == nil, "[fn] output num should not be zero")

	rst := reflect.MakeSlice(reflect.SliceOf(t), 0, l)
	for i := range values {
		val := values[i].Get()[0]
		if !val.IsValid() {
			val = reflect.Zero(t)
		}
		rst = reflect.Append(rst, val)
	}

	return rst.Interface()
}

func Promise(fn func(g xprocess_abc.Future)) xprocess_abc.IPromise {
	s := &promise{data: make(chan xprocess_abc.FutureValue)}
	s.wg.Inc()
	go func() {
		defer s.wg.Done()
		defer xerror.Resp(func(err xerror.XErr) {
			s.err.Store(err)
			s.Cancel()
		})
		fn(s)
	}()
	return s
}

func Async(fn interface{}, args ...interface{}) xprocess_abc.FutureValue {
	var err error
	var value = futureValueGet()
	var val = make(chan []reflect.Value)
	var vfn = xerror_util.FuncRaw(fn)
	go func() {
		defer xerror.Resp(func(err1 xerror.XErr) {
			err = xerror.WrapF(err1, "input:%#v, func:%s, caller:%s", args, reflect.TypeOf(fn), xerror_util.CallerWithFunc(fn))
			val <- nil
		})
		val <- vfn(args...)
	}()

	value.val = func() []reflect.Value { return <-val }
	value.err = func() error { return err }
	return value
}

func Await(val xprocess_abc.FutureValue, fn interface{}) xprocess_abc.FutureValue {
	var err error
	var value = futureValueGet()
	var values = make(chan []reflect.Value)
	var vfn = xerror_util.FuncValue(fn)
	go func() {
		v := val.Get()
		if err = val.Err(); err != nil {
			values <- nil
			return
		}

		defer xerror.Resp(func(err1 xerror.XErr) {
			err = xerror.WrapF(err1, "input:%#v, func:%#v", v, reflect.TypeOf(fn))
			values <- nil
		})
		values <- vfn(v...)
	}()
	value.val = func() []reflect.Value { return <-values }
	value.err = func() error { return err }
	return value
}

func valueStr(values ...reflect.Value) string {
	var data []interface{}
	for _, dt := range values {
		var val interface{}
		if dt.IsValid() {
			val = dt.Interface()
		}
		data = append(data, val)
	}
	return fmt.Sprint(data...)
}
