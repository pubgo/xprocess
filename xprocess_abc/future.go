package xprocess_abc

import "reflect"

type IPromise interface {
	Wait() error
	Cancelled() bool
	Value(fn interface{}) error // block
}

type Future interface {
	Cancel()
	Yield(data interface{}, fn ...interface{})   // async
	Await(val FutureValue, fn interface{}) error // block
}

type FutureValue interface {
	Err() error
	String() string
	Get() []reflect.Value
	Value(fn interface{}) error
}
