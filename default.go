package xprocess

import (
	"context"
	"time"

	"github.com/pubgo/xprocess/xprocess_test"
)

var defaultProcess = &process{}

func Test(fn interface{}) *xprocess_test.Test { return xprocess_test.TestFuncWith(fn) }
func MemStatsPrint()                          { defaultProcess.memStatsPrint() }
func CostWith(fn func()) time.Duration        { return defaultProcess.costWith(fn) }
func Count(n int) <-chan int                  { return defaultProcess.count(n) }

// Tick 简单定时器
// Example: Tick(100, time.Second)
func Tick(args ...interface{}) <-chan time.Time { return defaultProcess.tick(args...) }

// Try
// try wrap
func Try(fn func()) error { return defaultProcess.try(fn) }

// Go
// 启动一个goroutine
func Go(fn func(ctx context.Context)) context.CancelFunc { return defaultProcess.goCtx(fn) }

// GoLoop
// 启动一个goroutine loop
// 是为了替换 `go func() {for{ }}()` 这类的代码
func GoLoop(fn func(ctx context.Context)) context.CancelFunc { return defaultProcess.goLoopCtx(fn) }

// GoDelay
// 延迟goroutine
func GoDelay(dur time.Duration, fn func()) error { return defaultProcess.goWithDelay(dur, fn) }

// Timeout
// 执行超时函数, 超时后, 函数自动退出
func Timeout(dur time.Duration, fn func(ctx context.Context)) error {
	return defaultProcess.goWithTimeout(dur, fn)
}
