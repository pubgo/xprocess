package xprocess_abc

import "reflect"

type IPromise interface {
	RunComplete() error
	Await() chan FutureValue
	Map(fn interface{}) Value
}

type Future interface {
	Yield(data interface{})                  // async
	YieldFn(val FutureValue, fn interface{}) // block
}

type FutureValue interface {
	Err() error
	String() string
	Get() interface{}
	Raw() []reflect.Value
	Value(fn interface{}) error
}

type Value interface {
	Err() error
	String() string
	Value() interface{}
}
