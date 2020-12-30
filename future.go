package xprocess

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xerror/xerror_util"
	"go.uber.org/atomic"
)

type Future interface {
	Value(fn interface{})
	Cancelled() bool
	Wait() error
}

type Yield interface {
	Yield(data interface{})
	Await(val FutureValue, fn interface{})
	Cancel()
}

type future struct {
	wg        WaitGroup
	data      chan FutureValue
	done      sync.Once
	cancelled atomic.Bool
	err       atomic.Error
}

func (s *future) waitForClose()   { s.done.Do(func() { go func() { s.wg.Wait(); close(s.data) }() }) }
func (s *future) Wait() error     { s.wg.Wait(); return s.err.Load() }
func (s *future) Cancelled() bool { return s.cancelled.Load() }
func (s *future) Cancel()         { s.cancelled.Store(true) }
func (s *future) Await(v FutureValue, fn interface{}) {
	if s.cancelled.Load() {
		return
	}

	s.wg.Inc()
	go func() { defer s.wg.Done(); v.Value(fn) }()
}

func (s *future) Yield(data interface{}) {
	if s.cancelled.Load() {
		return
	}

	if val, ok := data.(FutureValue); ok {
		s.wg.Inc()
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

func (s *future) Value(fn interface{}) {
	s.waitForClose()

	vfn := reflect.ValueOf(fn)
	for data := range s.data {
		func() {
			defer futureValuePut(data.(*futureValue))
			defer s.wg.Done()
			defer xerror.RespRaise(func(err xerror.XErr) error {
				s.Cancel()
				return xerror.WrapF(err, xerror_util.CallerWithFunc(fn))
			})
			vfn.Call(data.Get())
		}()
	}
}

func Map(data interface{}, fn interface{}) interface{} {
	vfn := reflect.ValueOf(fn)
	vd := reflect.ValueOf(data)
	_l := vd.Len()
	var values []FutureValue
	for i := 0; i < _l; i++ {
		values = append(values, Async(vfn, vd.Index(i)))
	}

	var _t reflect.Type
	if vfn.Type().NumOut() > 0 {
		_t = vfn.Type().Out(0)
	}

	if _t == nil {
		return nil
	}

	var values1 = valueGet()
	for i := range values {
		val := values[i].Get()[0]
		if !val.IsValid() {
			val = reflect.Zero(_t)
		}
		values1 = append(values1, val)
	}

	_rst := reflect.MakeSlice(reflect.SliceOf(_t), 0, _l)
	for i := range values1 {
		_rst = reflect.Append(_rst, values1[i])
	}

	return _rst.Interface()
}

func Promise(fn func(y Yield)) Future {
	s := &future{data: make(chan FutureValue)}
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

func Async(fn interface{}, args ...interface{}) FutureValue {
	var err error
	var value = futureValueGet()
	var val = make(chan []reflect.Value)
	go func() {
		defer xerror.Resp(func(err1 xerror.XErr) {
			err = xerror.WrapF(err1, "input:%#v, func:%s, caller:%s", args, reflect.TypeOf(fn), xerror_util.CallerWithFunc(fn))
			val <- nil
		})
		val <- xerror_util.FuncRaw(fn)(args...)
	}()

	value.val = func() []reflect.Value { return <-val }
	value.err = func() error { return err }
	return value
}

func Await(val FutureValue, fn interface{}) FutureValue {
	var err error
	var value = futureValueGet()
	var values = make(chan []reflect.Value)

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
		values <- reflect.ValueOf(fn).Call(v)
	}()
	value.val = func() []reflect.Value { return <-values }
	value.err = func() error { return err }
	return value
}

var _valuePool = sync.Pool{
	New: func() interface{} {
		return []reflect.Value{}
	},
}

func valueGet() []reflect.Value {
	return _valuePool.Get().([]reflect.Value)
}

func valuePut(v []reflect.Value) {
	_valuePool.Put(v[:0])
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
