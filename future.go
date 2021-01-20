package xprocess

import (
	"github.com/pubgo/xprocess/xprocess_abc"
	"github.com/pubgo/xprocess/xprocess_future"
)

func Async(fn interface{}) func(args ...interface{}) xprocess_abc.FutureValue {
	return xprocess_future.Async(fn)
}

func Await(val xprocess_abc.FutureValue, fn interface{}) xprocess_abc.FutureValue {
	return xprocess_future.Await(val, fn)
}

func Promise(fn func(g xprocess_abc.Future)) xprocess_abc.IPromise { return xprocess_future.Promise(fn) }
func Map(data interface{}, fn interface{}) interface{}             { return xprocess_future.Map(data, fn) }
