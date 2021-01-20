package xprocess_future

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xerror/xerror_abc"
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

func (s *promise) Go(fn func()) {
	s.wg.Inc()
	go func() {
		defer s.wg.Done()
		defer xerror.Resp(func(err xerror.XErr) { s.Cancel(); s.err.Store(err) })

		fn()
	}()
}

func (s *promise) Await(val xprocess_abc.FutureValue, fn interface{}) error {
	if s.cancelled.Load() {
		return nil
	}

	return xerror.Wrap(val.Value(fn))
}

func (s *promise) waitForClose()   { s.done.Do(func() { go func() { s.wg.Wait(); close(s.data) }() }) }
func (s *promise) Wait() error     { s.wg.Wait(); return s.err.Load() }
func (s *promise) Cancelled() bool { return s.cancelled.Load() }
func (s *promise) Cancel()         { s.cancelled.Store(true) }
func (s *promise) Yield(data interface{}, fn ...interface{}) {
	if s.cancelled.Load() {
		return
	}

	// 处理xprocess_abc.FutureValue, 如果fn存在, 那么就调用Await
	if val, ok := data.(xprocess_abc.FutureValue); ok {
		if len(fn) > 0 {
			val = Await(val, fn[0])
		}

		s.wg.Inc()
		go func() { s.data <- val }()
		return
	}

	if val, ok := data.(func()); ok {
		s.Go(val)
		return
	}

	s.wg.Inc()
	value := futureValueGet()
	value.values = []reflect.Value{reflect.ValueOf(data)}
	go func() { s.data <- value }()
}

func (s *promise) Value(fn interface{}) (gErr error) {
	s.waitForClose()

	vfn := xerror_util.FuncValue(fn)
	for data := range s.data {
		func() {
			defer xerror.Resp(func(err xerror_abc.XErr) { s.Cancel(); gErr = xerror.WrapF(err, xerror_util.CallerWithFunc(fn)) })
			defer func() { futureValuePut(data.(*futureValue)); s.wg.Done() }()
			vfn(data.Get()...)
		}()
	}

	return
}

func Map(data interface{}, fn interface{}) interface{} {
	xerror.Assert(data == nil, "[data] should not be nil")
	xerror.Assert(fn == nil, "[fn] should not be nil")

	vfn := reflect.ValueOf(fn)
	vd := reflect.ValueOf(data)
	l := vd.Len()

	xerror.Assert(l == 0, "[fn] should not be nil")

	var values = make([]xprocess_abc.FutureValue, 0, l)
	for i := 0; i < l; i++ {
		values = append(values, Async(vfn)(vd.Index(i)))
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
	s.Go(func() { fn(s) })
	return s
}

func Async(fn interface{}) func(args ...interface{}) xprocess_abc.FutureValue {
	var vfn = xerror_util.FuncRaw(fn)
	return func(args ...interface{}) xprocess_abc.FutureValue {
		var err error
		var value = futureValueGet()
		var val = make(chan []reflect.Value)

		go func() {
			defer xerror.Resp(func(err1 xerror.XErr) {
				err = err1.WrapF(err1, "recovery error, input:%#v, func:%s, caller:%s", args, reflect.TypeOf(fn), xerror_util.CallerWithFunc(fn))
				val <- []reflect.Value{}
			})
			val <- vfn(args...)
		}()

		value.val = func() []reflect.Value { return <-val }
		value.err = func() error { return err }
		return value
	}
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
