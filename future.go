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
	Raw() ([]reflect.Value, error)
	Value(fn ...interface{})
	Get() interface{}
}

var _ Value = (*valueFunc)(nil)

type valueFunc struct {
	err func() error
	val func() []reflect.Value
}

func (v valueFunc) Raw() ([]reflect.Value, error) {
	return v.val(), v.err()
}

func (v valueFunc) Value(fn ...interface{}) {
	xerror.Next().Panic(v.err())
	reflect.ValueOf(fn).Call(v.val())
}

func (v valueFunc) Get() interface{} {
	xerror.Next().Panic(v.err())
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
		if val, ok := val.(Value); ok {
			defer xerror.RespRaise("valueFunc.Get")
			return val.Get()
		}
	}
	return nil
}

type IFuture interface {
	Value(fn interface{})
	Chan() <-chan interface{}
}

type Yield interface {
	Yield(data interface{})
	Await(fn interface{}, args ...interface{}) func(func(fn interface{}))
}

type future struct {
	wg   sync.WaitGroup
	num  int32
	data chan Value
	done sync.Once
}

func (s *future) Await(fn interface{}, args ...interface{}) func(func(fn interface{})) {
	panic("implement me")
}

func (s *future) Value(fn interface{}) {
	vfn := reflect.ValueOf(fn)
	for data := range s.data {
		val := xerror.PanicErr(data.Raw())
		vfn.Call([]reflect.Value{reflect.ValueOf(val)})
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

func (s *future) Async(fn interface{}, args ...interface{}) {
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

	var val []reflect.Value
	var errChan = make(chan error)
	s.Go(func() {
		defer xerror.Resp(func(err xerror.XErr) { errChan <- err })
		val = vfn.Call(values)
		//if len(dt) == 0 {
		//	xerror.Panic(xerror.New("output num is zero"))
		//}

		//if len(dt) > 0 {
		//	val = dt[0].Interface()
		//}

		//if len(dt) > 1 && dt[1].IsValid() && !dt[1].IsNil() {
		//	xerror.Panic(dt[1].Interface().(error))
		//}

		errChan <- nil
	})

	s.Yield(&valueFunc{err: func() error { return <-errChan }, val: func() []reflect.Value { return val }})
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

	var val []reflect.Value
	var errChan = make(chan error)
	go func() {
		defer xerror.Resp(func(err xerror.XErr) { errChan <- err })

		val = vfn.Call(values)
		//if len(dt) == 0 {
		//	xerror.Panic(xerror.New("output num is zero"))
		//}
		//
		//if len(dt) > 0 {
		//	val = dt[0].Interface()
		//}
		//
		//if len(dt) > 1 && dt[1].IsValid() && !dt[1].IsNil() {
		//	xerror.Panic(dt[1].Interface().(error))
		//}

		errChan <- nil
	}()

	return &valueFunc{val: func() []reflect.Value { return val }, err: func() error { return <-errChan }}
}

func Await1(val Value, fn ...interface{}) { val.Value(fn...) }
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
