package xprocess

import (
	"context"
	"encoding/json"
	"github.com/pubgo/xerror"
	"github.com/pubgo/xerror/xerror_util"
	"go.uber.org/atomic"
	"reflect"
)

var defaultProcess = &process{}

func Go(fn func(ctx context.Context) error) func() error {
	return defaultProcess.goCtx(fn)
}

func GoLoop(fn func(ctx context.Context) error) func() error {
	return defaultProcess.goLoopCtx(fn)
}

func Stack() string {
	var _data = make(map[string]int32)
	data.Range(func(key, value interface{}) bool {
		_data[xerror_util.CallerWithFunc(key.(reflect.Value).Interface())] = value.(*atomic.Int32).Load()
		return true
	})
	dt := xerror.PanicBytes(json.Marshal(_data))
	return string(dt)
}
