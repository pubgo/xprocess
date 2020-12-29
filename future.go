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

type IFuture interface {
	Value(fn interface{})
}

type Yield interface {
	Yield(data interface{})
	Await(val Value, fn ...interface{})
}

type future struct {
	wg   sync.WaitGroup
	data chan Value
	done sync.Once
}

func (s *future) inc() { s.wg.Add(1) }
func (s *future) Await(val Value, pipe ...interface{}) {
	s.inc()
	if len(pipe) > 0 {
		val = val.pipe(pipe[0])
	}
	go func() { s.data <- val }()
}

func (s *future) Yield(data interface{}) {
	if val, ok := data.(Value); ok {
		s.Await(val)
		return
	}

	s.inc()
	go func() {
		s.data <- &futureValue{val: func() []reflect.Value { return []reflect.Value{reflect.ValueOf(data)} }}
	}()
}

func (s *future) close() { s.done.Do(func() { go func() { s.wg.Wait(); close(s.data) }() }) }
func (s *future) Value(fn interface{}) {
	s.close()

	vfn := reflect.ValueOf(fn)
	for data := range s.data {
		go func(val Value) { defer s.wg.Done(); vfn.Call(val.rawValues()) }(data)
	}
}

func Future(fn func(y Yield)) IFuture {
	s := &future{data: make(chan Value)}
	s.inc()
	go func() { defer s.wg.Done(); fn(s) }()
	return s
}

func Async(fn interface{}, args ...interface{}) Value {
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
	go func() {
		defer xerror.Resp(func(err xerror.XErr) { xlog.Error("Async panic", xlog.Any("err", err)) })
		val <- vfn.Call(values)
	}()
	return &futureValue{val: func() []reflect.Value { return <-val }}
}

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
