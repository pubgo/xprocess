package xprocess

import (
	"github.com/pubgo/xprocess/xprocess_abc"
	"github.com/pubgo/xprocess/xprocess_future"
)

func AwaitFn(fn interface{}, args ...interface{}) xprocess_abc.FutureValue {
	return xprocess_future.AwaitFn(fn, args...)
}

func Await(val xprocess_abc.FutureValue, fn interface{}) xprocess_abc.FutureValue {
	return xprocess_future.Await(val, fn)
}

func Promise(fn func(g xprocess_abc.Future)) xprocess_abc.IPromise { return xprocess_future.Promise(fn) }
