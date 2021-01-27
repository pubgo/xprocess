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

var _ xprocess_abc.Future = (*future)(nil)

type future struct{ p *promise }

func (f future) Cancel()                                              { f.p.cancel() }
func (f future) Yield(data interface{})                               { f.p.yield(data) }
func (f future) YieldFn(val xprocess_abc.FutureValue, fn interface{}) { f.p.await(val, fn) }

type promise struct {
	wg        xprocess_waitgroup.WaitGroup
	data      chan xprocess_abc.FutureValue
	done      sync.Once
	cancelled atomic.Bool
	err       atomic.Error
}

func (s *promise) cancel()         { s.cancelled.Store(true) }
func (s *promise) waitForClose()   { s.done.Do(func() { go func() { s.wg.Wait(); close(s.data) }() }) }
func (s *promise) Cancelled() bool { return s.cancelled.Load() }
func (s *promise) Await() chan xprocess_abc.FutureValue {
	s.waitForClose()

	var val = make(chan xprocess_abc.FutureValue)
	go func() {
		defer close(val)
		for data := range s.data {
			val <- data
			s.wg.Done()
		}
	}()

	return val
}

func (s *promise) yield(data interface{}) {
	if s.cancelled.Load() {
		return
	}

	if data == nil {
		s.wg.Inc()
		val := futureValueGet()
		val.val <- nil
		go func() { s.data <- val }()
		return
	}

	if val, ok := data.(func()); ok {
		s.Go(val)
		return
	}

	if val, ok := data.(*futureValue); ok {
		s.wg.Inc()
		go func() { s.data <- val }()
		return
	}

	s.wg.Inc()
	value := futureValueGet()
	value.val <- []reflect.Value{reflect.ValueOf(data)}
	go func() { s.data <- value }()

}

func (s *promise) await(val xprocess_abc.FutureValue, fn interface{}) {
	if s.cancelled.Load() {
		return
	}

	s.yield(Await(val, fn))
}

func (s *promise) RunUntilComplete() (err error) {
	s.waitForClose()

	defer xerror.RespErr(&err)

	var errs []error
	for data := range s.Await() {
		if err := data.Err(); err != nil {
			errs = append(errs, err)
		}
		futureValuePut(data.(*futureValue))
	}

	xerror.PanicCombine(errs...)
	return nil
}

func (s *promise) Map(fn interface{}) (val1 xprocess_abc.Value) {
	s.waitForClose()

	defer xerror.Resp(func(err xerror.XErr) { val1 = &value{err: err} })

	xerror.Assert(fn == nil, "[fn] should not be nil")

	var errs []error
	var values []xprocess_abc.FutureValue
	for data := range s.Await() {
		if err := data.Err(); err != nil {
			errs = append(errs, err)
		}

		values = append(values, Await(data, fn))
	}

	xerror.PanicCombine(errs...)

	if len(values) == 0 {
		return
	}

	var tfn = reflect.TypeOf(fn)

	// fn没有输出
	if tfn.NumOut() == 0 {
		xerror.Panic(s.RunUntilComplete())
	}

	var t = tfn.Out(0)
	var rst = reflect.MakeSlice(reflect.SliceOf(t), 0, len(values))
	for i := range values {
		if err := values[i].Err(); err != nil {
			errs = append(errs, err)
			continue
		}

		val := values[i].Raw()[0]
		if !val.IsValid() {
			val = reflect.Zero(t)
		}
		rst = reflect.Append(rst, val)

		futureValuePut(values[i].(*futureValue))
	}
	xerror.PanicCombine(errs...)

	return &value{val: rst.Interface()}
}

func (s *promise) Go(fn func()) {
	s.wg.Inc()
	go func() {
		defer s.wg.Done()
		defer xerror.Resp(func(err xerror.XErr) { s.cancel(); s.err.Store(err) })
		fn()
	}()
}

func Promise(fn func(g xprocess_abc.Future)) xprocess_abc.IPromise {
	s := &promise{data: make(chan xprocess_abc.FutureValue)}
	s.Go(func() { fn(future{p: s}) })
	return s
}

func Async(fn interface{}, args ...interface{}) (val1 xprocess_abc.FutureValue) {
	var value = futureValueGet()

	defer xerror.Resp(func(err xerror.XErr) {
		value.err = err
		val1 = value
	})

	xerror.Assert(fn == nil, "[fn] should not be nil")

	var vfn = xerror_util.FuncRaw(fn)

	go func() {
		defer xerror.Resp(func(err1 xerror.XErr) {
			value.val <- []reflect.Value{}
			value.err = err1.WrapF("recovery error, input:%#v, func:%s, caller:%s", args, reflect.TypeOf(fn), xerror_util.CallerWithFunc(fn))
		})

		value.val <- vfn(args...)
	}()

	return value
}

func Await(val xprocess_abc.FutureValue, fn interface{}) (val1 xprocess_abc.FutureValue) {

	var value = futureValueGet()

	defer xerror.Resp(func(err xerror.XErr) { val1 = value.setErr(err) })

	xerror.Assert(val == nil, "[val] should not be nil")
	xerror.Assert(fn == nil, "[fn] should not be nil")

	var vfn = xerror_util.FuncValue(fn)
	go func() {
		if err := val.Err(); err != nil {
			value.setErr(err)
			return
		}

		defer xerror.Resp(func(err1 xerror.XErr) {
			value.setErr(err1.WrapF("input:%#v, func:%#v", val.Get(), reflect.TypeOf(fn)))
		})

		value.val <- vfn(val.Raw()...)
	}()

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
