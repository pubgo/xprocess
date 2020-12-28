package xprocess

import (
	"reflect"
	"runtime"
	"sync"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xerror/xerror_util"
	"github.com/pubgo/xlog"
)

var ErrInputOutputParamsNotMatch = xerror.New("the input num and output num of the callback func is not match")
var ErrFuncOutputTypeNotMatch = xerror.New("the  output type of the callback func is not match")
var ErrCallBackInValid = xerror.New("the func is invalid")

type Value interface {
	Raw() []reflect.Value    // block
	Value(fn ...interface{}) // async callback
	Get() interface{}        // wait for value
}

var _ Value = (*valueFunc)(nil)

type valueFunc struct {
	val func() []reflect.Value
}

func (v valueFunc) Raw() []reflect.Value {
	return v.val()
}

func (v valueFunc) Value(fn ...interface{}) {
	if len(fn) == 0 {
		return
	}

	go func() { reflect.ValueOf(fn[0]).Call(v.val()) }()
}

func (v valueFunc) Get() interface{} {
	values := v.val()
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
	Chan() <-chan interface{}
}

type Yield interface {
	Yield(data interface{})
	Await(val Value, fn ...interface{})
}

type future struct {
	wg   sync.WaitGroup
	num  int32
	data chan Value
	done sync.Once
}

func (s *future) Await(val Value, fn ...interface{}) { val.Value(fn...) }
func (s *future) Value(fn interface{}) {
	vfn := reflect.ValueOf(fn)
	for data := range s.data {
		go func(val Value) { vfn.Call(val.Raw()) }(data)
	}
}

func (s *future) Yield(data interface{}) {
	dt, ok := data.(Value)
	if ok {
		s.data <- dt
		return
	}

	s.data <- &valueFunc{val: func() []reflect.Value { return []reflect.Value{reflect.ValueOf(data)} }}
}

func (s *future) Chan() <-chan interface{} {
	var data = make(chan interface{})
	go func() {
		defer close(data)
		for dt := range s.data {
			data <- dt
		}
	}()
	return data
}

func (s *future) Go(fn func()) {
	s.wg.Add(1)

	s.done.Do(func() {
		go func() {
			s.wg.Wait()
			close(s.data)
		}()
	})

	go func() {
		defer s.wg.Done()
		fn()
	}()
}

func Future(fn func(y Yield), nums ...int) IFuture {
	num := runtime.NumCPU() * 2
	if len(nums) > 0 {
		num = nums[0]
	}

	s := &future{num: int32(num), data: make(chan Value, num)}
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
