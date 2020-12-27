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

type IFuture interface {
	Wait()
	Await(func(data interface{}))
	Chan() <-chan interface{}
	Future(fn func(y Yield), nums ...int) IFuture
}

type Yield interface {
	Go(fn func())
	Return(data interface{})
}

type future struct {
	num    int32
	count  atomic.Int32
	data   chan interface{}
	stop   *atomic.Bool
	done   chan bool
	mu     sync.Mutex
	cancel []func()
}

func (s *future) Wait() {
	//s.wg.Wait()
	//s.stop.Store(true)
}

func (s *future) Future(fn func(y Yield), nums ...int) IFuture {
	num := runtime.NumCPU() * 2
	if len(nums) > 0 {
		num = nums[0]
	}

	stm := &future{stop: s.stop, num: int32(num), data: make(chan interface{}, num)}

	go func() {
		defer func() {
			<-s.done
			close(s.data)
		}()

		defer xerror.Resp(func(err xerror.XErr) {
			xlog.Debug("Future panic", xlog.Any("err", err))
			stm.stop.Store(true)
		})

		fn(stm)
	}()
	return stm
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

func (s *future) Return(data interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stop.Load() {
		return
	}

	s.data <- data
}

func (s *future) Await(fn func(data interface{})) {
	for data := range s.data {
		fn(data)
	}
}

func (s *future) Go(fn func()) {
	if s.stop.Load() {
		return
	}

	s.count.Inc()
	go func() {
		defer func() {
			s.count.Dec()
			fmt.Println("s.count", s.count.Load())
		}()

		defer xerror.Resp(func(err xerror.XErr) {
			xlog.Debug("future.Go panic", xlog.Any("err", err))
			s.stop.Store(true)
		})

		fn()
	}()
}

func Future(fn func(y Yield), nums ...int) IFuture {
	num := runtime.NumCPU() * 2
	if len(nums) > 0 {
		num = nums[0]
	}

	s := &future{stop: atomic.NewBool(false), num: int32(num), done: make(chan bool), data: make(chan interface{}, num)}
	go func() {
		defer func() {
			<-s.done
			close(s.data)
		}()

		defer xerror.Resp(func(err xerror.XErr) {
			xlog.Debug("Future panic", xlog.Any("err", err))
			s.stop.Store(true)
		})

		fn(s)
	}()

	return s
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
