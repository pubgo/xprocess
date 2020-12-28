package xprocess

import (
	"fmt"
	"github.com/pubgo/xerror"
	"github.com/pubgo/xerror/xerror_util"
	"github.com/pubgo/xlog"
	"go.uber.org/atomic"
	"reflect"
	"runtime"
	"sync"
)

var ErrInputOutputParamsNotMatch = xerror.New("the input num and output num of the callback func is not match")
var ErrFuncOutputTypeNotMatch = xerror.New("the  output type of the callback func is not match")
var ErrCallBackInValid = xerror.New("the func is invalid")

type Value interface {
	Raw() []reflect.Value         // block
	Value(fn ...interface{})      // async callback
	Pipe(fn ...interface{}) Value // async callback
	Get() interface{}             // block wait for value
}

var _ Value = (*valueFunc)(nil)

type valueFunc struct {
	val    func() []reflect.Value
	values []reflect.Value
	done   sync.Once
}

func (v *valueFunc) Pipe(fn ...interface{}) Value {
	if len(fn) == 0 {
		return v
	}

	var values = make(chan []reflect.Value)
	go func() {
		defer xerror.Resp(func(err xerror.XErr) { xlog.Error("Pipe panic", xlog.Any("err", err)) })
		values <- reflect.ValueOf(fn[0]).Call(v.getVal())
	}()

	return &valueFunc{val: func() []reflect.Value { return <-values }}
}

func (v *valueFunc) getVal() []reflect.Value {
	v.done.Do(func() { v.values = v.val() })
	return v.values
}

func (v *valueFunc) Raw() []reflect.Value {
	return v.getVal()
}

func (v *valueFunc) Value(fn ...interface{}) {
	if len(fn) == 0 {
		return
	}

	reflect.ValueOf(fn[0]).Call(v.getVal())
}

func (v *valueFunc) Get() interface{} {
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
			defer xerror.RespRaise("valueFunc.Get")
			return val1.Get()
		}
		return val
	}

	return nil
}

type IFuture interface {
	Value(fn interface{})
}

type Yield interface {
	Yield(data interface{})
	Await(val Value, fn ...interface{})
}

type future struct {
	wg    *sync.WaitGroup
	data  chan Value
	num   int32
	done  sync.Once
	count atomic.Int32
}

func (s *future) Await(val Value, pipe ...interface{}) {
	s.wg.Add(1)
	s.count.Inc()
	if len(pipe) > 0 {
		val = val.Pipe(pipe...)
	}
	go func() { s.data <- val }()
}

func (s *future) Yield(data interface{}) {
	if val, ok := data.(Value); ok {
		s.Await(val)
		return
	}

	s.wg.Add(1)
	s.count.Inc()

	val := &valueFunc{val: func() []reflect.Value { return []reflect.Value{reflect.ValueOf(data)} }}
	go func() { s.data <- val }()
}

func (s *future) close() {
	s.done.Do(func() {
		go func() {
			for {
				s.wg.Wait()
				fmt.Println(s.count.Load())
				if s.count.Load() == 0 {
					break
				}
			}

			close(s.data)
		}()
	})
}

func (s *future) Value(fn interface{}) {
	vfn := reflect.ValueOf(fn)
	for data := range s.data {
		s.close()

		go func(val Value) {
			defer s.wg.Done()
			defer s.count.Dec()
			vfn.Call(val.Raw())
		}(data)
	}
}

func Future(fn func(y Yield), nums ...int) IFuture {
	num := runtime.NumCPU() * 2
	if len(nums) > 0 {
		num = nums[0]
	}

	s := &future{num: int32(num), data: make(chan Value, num), wg: &sync.WaitGroup{}}
	go fn(s)
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
	return &valueFunc{val: func() []reflect.Value { return <-val }}
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
