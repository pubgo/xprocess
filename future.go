package xprocess

import (
	"reflect"
	"runtime"
	"sync"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xerror/xerror_util"
)

var ErrInputOutputParamsNotMatch = xerror.New("the input num and output num of the callback func is not match")
var ErrFuncOutputTypeNotMatch = xerror.New("the  output type of the callback func is not match")
var ErrCallBackInValid = xerror.New("the func is invalid")

type Value interface {
	Value() interface{}
	Err() error
}

func NewValue(val interface{}, err error) Value {
	return ValueSs1{val: val, err: err}
}

var _ Value = (*ValueSs1)(nil)

type ValueSs1 struct {
	err error
	val interface{}
}

func (v ValueSs1) Value() interface{} {
	return v.val
}

func (v ValueSs1) Err() error {
	return v.err
}

var _ Value = (*ValueSs)(nil)

type ValueSs struct {
	err chan error
	val chan interface{}
}

func (v ValueSs) Value() interface{} {
	return <-v.val
}

func (v ValueSs) Err() error {
	return <-v.err
}

type IFuture interface {
	Await(fn func(data Value))
}

type Yield interface {
	Go(fn func())
	Return(data Value)
}

type future struct {
	wg   sync.WaitGroup
	num  int32
	data chan Value
	done sync.Once
}

func (s *future) Yield(fn interface{}, args ...interface{}) (err error) {
	defer xerror.RespExit()

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

	var dt []reflect.Value
	var errChan = make(chan error)
	s.Go(func() {
		defer xerror.Resp(func(err xerror.XErr) { errChan <- err })
		dt = vfn.Call(values)
		errChan <- nil
	})
	xerror.Panic(<-errChan)

	if len(dt) == 0 {
		return xerror.New("output num is zero")
	}

	if len(dt) > 0 {
		//s.Return(dt[0].Interface())
	}

	if len(dt) > 1 && dt[1].IsValid() && !dt[1].IsNil() {
		return dt[1].Interface().(error)
	}

	return nil
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

func (s *future) Return(data Value) {
	s.data <- data
}

func (s *future) Await(fn func(data Value)) {
	for data := range s.data {
		fn(data)
	}
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

	var val = make(chan interface{})
	var errChan = make(chan error)
	go func() {
		defer xerror.Resp(func(err xerror.XErr) {
			errChan <- err
			val <- nil
		})

		dt := vfn.Call(values)
		if len(dt) == 0 {
			xerror.Panic(xerror.New("output num is zero"))
		}

		if len(dt) > 0 {
			val <- dt[0].Interface()
		}

		if len(dt) > 1 && dt[1].IsValid() && !dt[1].IsNil() {
			xerror.Panic(dt[1].Interface().(error))
		}

		errChan <- nil
	}()

	return &ValueSs{val: val, err: errChan}
}

func Await(fn interface{}, args ...interface{}) func(fn ...interface{}) {
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

	var ret = make(chan []reflect.Value)
	var err error
	go func() {
		defer xerror.RespErr(&err)
		ret <- vfn.Call(values)
	}()

	var dt = <-ret
	xerror.Next().Panic(err)

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

		cfn.Call(dt)
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
