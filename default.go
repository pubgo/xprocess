package xprocess

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xerror/xerror_util"
	"go.uber.org/atomic"
)

var defaultProcess = &process{}

// Go
// 启动一个goroutine
func Go(fn func(ctx context.Context)) context.CancelFunc {
	return defaultProcess.goCtx(fn)
}

// GoLoop
// 启动一个goroutine loop
// 是为了替换 `go func() {for{ }}()` 这类的代码
func GoLoop(fn func(ctx context.Context)) context.CancelFunc {
	return defaultProcess.goLoopCtx(fn)
}

// Timeout
// 执行超时函数, 超时后, 函数自动退出
func Timeout(dur time.Duration, fn func(ctx context.Context) error) error {
	return defaultProcess.goWithTimeout(dur, fn)
}

// Stack
// 获取正在运行的goroutine的stack和数量
func Stack() string {
	var _data = make(map[string]int32)
	data.Range(func(key, value interface{}) bool {
		_data[xerror_util.CallerWithFunc(key.(reflect.Value).Interface())] = value.(*atomic.Int32).Load()
		return true
	})
	dt := xerror.PanicBytes(json.Marshal(_data))
	return string(dt)
}
