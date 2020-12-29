package xprocess

import (
	"reflect"
	"sync"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xerror/xerror_util"
	"github.com/pubgo/xlog"
)

var ErrInputOutputParamsNotMatch = xerror.New("the input num and output num of the callback func is not match")
var ErrFuncOutputTypeNotMatch = xerror.New("the  output type of the callback func is not match")
var ErrCallBackInValid = xerror.New("the func is invalid")

type Future interface {
	Err(func(err error)) Future
	Value(fn interface{}) Future
	Cancelled() bool
	Done() bool
	Chan() <-chan interface{}
}

type Yield interface {
	Yield(data interface{})
	Await(val FutureValue, fn interface{})
	Cancel()
}

type future struct {
	wg      WaitGroup
	data    chan FutureValue
	done    sync.Once
	errCall func(err error)
}

func (s *future) Await(val FutureValue, fn interface{}) { val.Value(fn) }
func (s *future) Err(f func(err error)) Future          { s.errCall = f; return s }
func (s *future) Cancelled() bool                       { return true }
func (s *future) Done() bool                            { return true }
func (s *future) cancel()                               {}
func (s *future) Cancel()                               { s.waitForClose() }
func (s *future) Yield(data interface{}) {
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
				xlog.Error("future.Yield panic", xlog.Any("err", err))
				s.Cancel()
			})

			val()
		}()
		return
	}

	s.wg.Inc()
	go func() { s.data <- &futureValue{val: func() []reflect.Value { return []reflect.Value{reflect.ValueOf(data)} }} }()
}

func (s *future) waitForClose() { s.done.Do(func() { go func() { s.wg.Wait(); close(s.data) }() }) }
func (s *future) Chan() <-chan interface{} {
	s.waitForClose()

	var data = make(chan interface{})
	go func() {
		for val := range s.data {
			val.Err(s.errCall)
			s.wg.Done()
			data <- val.Get()
		}
	}()
	return data
}

func (s *future) Value(fn interface{}) Future {
	s.waitForClose()

	vfn := reflect.ValueOf(fn)
	for data := range s.data {
		go func(val FutureValue) { defer s.wg.Done(); vfn.Call(val.raw()) }(data)
	}

	return s
}

func Promise(fn func(y Yield)) Future {
	s := &future{data: make(chan FutureValue)}
	s.wg.Inc()
	go func() {
		defer s.wg.Done()
		defer xerror.RespExit()
		fn(s)
	}()
	return s
}

func Async(fn interface{}, args ...interface{}) FutureValue {
	vfn := reflect.ValueOf(fn)
	if vfn.Kind() != reflect.Func {
		xerror.Next().Panic(xerror.New("[fn] type should be func"))
	}

	if !vfn.IsValid() || vfn.IsNil() {
		xerror.Next().Panic(xerror.New("[fn] should not be nil"))
	}

	var values = valueGet()
	defer valuePut(values)

	for _, k := range args {
		values = append(values, reflect.ValueOf(k))
	}

	for i, k := range values {
		if !k.IsValid() {
			values[i] = reflect.New(vfn.Type().In(i)).Elem()
			continue
		}

		switch k.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
			if k.IsNil() {
				values[i] = reflect.New(vfn.Type().In(i)).Elem()
				continue
			}
		}
	}

	var val = make(chan []reflect.Value)
	var err error
	go func() {
		defer xerror.Resp(func(err1 xerror.XErr) { err = err1; val <- nil })
		val <- vfn.Call(values)
	}()
	return &futureValue{val: func() []reflect.Value { return <-val }, err: func() error { return err }}
}

func Await(val FutureValue, fn interface{}) FutureValue { return val.pipe(fn) }

func _Await(fn interface{}, args ...interface{}) func(fn ...interface{}) {
	vfn := reflect.ValueOf(fn)

	var values = valueGet()
	defer valuePut(values)

	for _, k := range args {
		values = append(values, reflect.ValueOf(k))
	}

	for i, k := range values {
		if !k.IsValid() {
			args[i] = reflect.New(vfn.Type().In(i)).Elem()
			continue
		}

		switch k.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.UnsafePointer:
			if k.IsNil() {
				args[i] = reflect.New(vfn.Type().In(i)).Elem()
				continue
			}
		}

		values[i] = k
	}

	return func(fn ...interface{}) {
		if len(fn) == 0 {
			return
		}

		var cfn reflect.Value
		cfn = reflect.ValueOf(fn[0])
		if !cfn.IsValid() || cfn.IsZero() {
			xerror.Next().Panic(ErrCallBackInValid)
		}

		if cfn.Type().NumIn() != vfn.Type().NumOut() {
			xerror.Next().PanicF(ErrInputOutputParamsNotMatch, "fn:%s, [%d]--[%d]", xerror_util.CallerWithFunc(fn[0]), cfn.Type().NumIn(), vfn.Type().NumOut())
		}

		if cfn.Type().NumIn() != 0 && cfn.Type().In(0) != vfn.Type().Out(0) {
			xerror.Next().PanicF(ErrFuncOutputTypeNotMatch, "fn:%s, [%s]--[%s]", xerror_util.CallerWithFunc(fn[0]), cfn.Type().In(0), vfn.Type().Out(0))
		}

		go func() {
			defer xerror.Resp(func(err xerror.XErr) { xlog.Error("Await panic", xlog.Any("err", err)) })
			val := vfn.Call(values)
			cfn.Call(val)
		}()
	}
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
