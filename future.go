package xprocess

import (
	"github.com/pubgo/xprocess/xprocess_abc"
	"github.com/pubgo/xprocess/xprocess_future"
)

type FutureValue = xprocess_abc.FutureValue
type IPromise = xprocess_abc.IPromise
type Value = xprocess_abc.Value

func Async(fn interface{}, args ...interface{}) xprocess_abc.FutureValue {
	return xprocess_future.Async(fn, args...)
}

func Await(val xprocess_abc.FutureValue, fn interface{}) xprocess_abc.FutureValue {
	return xprocess_future.Await(val, fn)
}

func Promise(fn func(g xprocess_abc.Future)) xprocess_abc.IPromise { return xprocess_future.Promise(fn) }
