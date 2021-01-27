package xprocess_utils

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xprocess/xprocess_strings"
)

type frame uintptr

func (f frame) pc() uintptr { return uintptr(f) - 1 }

// CallerStack 调用栈
func CallerStack(cd int, fns ...func(fn *runtime.Func, pc uintptr) string) string {
	var pcs = make([]uintptr, 1)
	if runtime.Callers(cd, pcs[:]) == 0 {
		return ""
	}

	f := frame(pcs[0])
	fn := runtime.FuncForPC(f.pc())
	if fn == nil {
		return "unknown type"
	}

	if len(fns) > 0 {
		return fns[0](fn, f.pc())
	}

	file, line := fn.FileLine(f.pc())
	return xprocess_strings.Append(file, ":", strconv.Itoa(line))
}

// FuncStack 函数栈
func FuncStack(fn interface{}, fns ...func(fn *runtime.Func, pc uintptr) string) string {
	xerror.Assert(fn == nil, "[fn] is nil")

	var vfn = reflect.ValueOf(fn)
	xerror.Assert(!vfn.IsValid() || vfn.Kind() != reflect.Func || vfn.IsNil(), "[fn] func is nil or type error")

	var fnStack = runtime.FuncForPC(vfn.Pointer())
	if len(fns) > 0 {
		fns[0](fnStack, vfn.Pointer())
	}

	var file, line = fnStack.FileLine(vfn.Pointer())
	ma := strings.Split(fnStack.Name(), ".")
	return xprocess_strings.Append(file, ":", strconv.Itoa(line), " ", ma[len(ma)-1])
}

func FuncValue(fn interface{}) func(...reflect.Value) []reflect.Value {
	xerror.Assert(fn == nil, "[fn] is nil")

	vfn, ok := fn.(reflect.Value)
	if !ok {
		vfn = reflect.ValueOf(fn)
	}

	xerror.Assert(!vfn.IsValid() || vfn.Kind() != reflect.Func || vfn.IsNil(), "[fn] type error or nil")

	var tfn = vfn.Type()
	var numIn = tfn.NumIn()
	var variadicType reflect.Type
	var variadicValue reflect.Value
	if tfn.IsVariadic() {
		variadicType = tfn.In(numIn - 1)
		variadicValue = reflect.Zero(variadicType)
		if isElem(variadicType.Kind()) {
			variadicValue = reflect.New(variadicType).Elem()
		}
	}

	return func(args ...reflect.Value) []reflect.Value {
		xerror.Assert(variadicType == nil && numIn != len(args) || variadicType != nil && len(args) < numIn-1,
			"the input params of func is not match, func: %s, numIn:%d numArgs:%d\n", tfn, numIn, len(args))

		for i := range args {
			// variadic
			if i >= numIn && !args[i].IsValid() {
				args[i] = variadicValue
				continue
			}

			if !args[i].IsValid() {
				args[i] = reflect.Zero(tfn.In(i))
				if isElem(args[i].Kind()) && args[i].IsNil() {
					args[i] = reflect.New(tfn.In(i).Elem())
				}
				continue
			}

			if isElem(args[i].Kind()) && args[i].IsNil() {
				args[i] = reflect.Zero(tfn.In(i))
			}
		}

		defer xerror.RespRaise(func(err xerror.XErr) error {
			valuePut(args)
			return err.WrapF("[vfn.Call] panic, err:%#v, args:%s, fn:%s", err, valueStr(args...), tfn)
		})

		return vfn.Call(args)
	}
}

func FuncRaw(fn interface{}) func(...interface{}) []reflect.Value {
	vfn := FuncValue(fn)
	return func(args ...interface{}) []reflect.Value {
		var args1 = valueGet()
		for i := range args {
			var vk reflect.Value
			if args[i] == nil {
				vk = reflect.ValueOf(args[i])
			} else if k1, ok := args[i].(reflect.Value); ok {
				vk = k1
			} else {
				vk = reflect.ValueOf(args[i])
			}
			args1 = append(args1, vk)
		}
		return vfn(args1...)
	}
}

func Func(fn interface{}) func(...interface{}) func(...interface{}) {
	vfn := FuncRaw(fn)
	return func(args ...interface{}) func(...interface{}) {
		ret := vfn(args...)
		return func(fns ...interface{}) {
			if len(fns) == 0 {
				return
			}

			xerror.Assert(fns[0] == nil, "[fns] is nil")

			cfn, ok := fns[0].(reflect.Value)
			if !ok {
				cfn = reflect.ValueOf(fns[0])
			}
			xerror.Assert(!cfn.IsValid() || cfn.Kind() != reflect.Func || cfn.IsNil(),
				"[fns] type error or nil")

			tfn := reflect.TypeOf(fn)
			xerror.Assert(cfn.Type().NumIn() != tfn.NumOut(),
				"the input num and output num of the callback func is not match, [%d]<->[%d]\n",
				cfn.Type().NumIn(), tfn.NumOut())

			xerror.Assert(cfn.Type().NumIn() != 0 && cfn.Type().In(0) != tfn.Out(0),
				"the output type of the callback func is not match, [%s]<->[%s]\n",
				cfn.Type().In(0), tfn.Out(0))

			defer xerror.RespRaise(func(err xerror.XErr) error {
				valuePut(ret)
				return err.WrapF("[cfn.Call] panic, err:%#v, args:%s, fn:%s", err, valueStr(ret...), cfn.Type())
			})

			cfn.Call(ret)
		}
	}
}

var valuePool = sync.Pool{New: func() interface{} { return make([]reflect.Value, 0, 1) }}

func valueGet() []reflect.Value  { return valuePool.Get().([]reflect.Value) }
func valuePut(v []reflect.Value) { valuePool.Put(v[:0]) }

func valueStr(values ...reflect.Value) string {
	var data []interface{}
	for i := range values {
		var val interface{} = nil
		if values[i].IsValid() {
			val = values[i].Interface()
		}
		data = append(data, val)
	}
	return fmt.Sprint(data...)
}

func isElem(val reflect.Kind) bool {
	switch val {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return true
	}
	return false
}
