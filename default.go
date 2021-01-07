package xprocess

import (
	"context"
	"time"
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

// GoDelay
// 延迟goroutine
func GoDelay(dur time.Duration, fn func(ctx context.Context)) context.CancelFunc {
	return defaultProcess.goWithDelay(dur, fn)
}

// Timeout
// 执行超时函数, 超时后, 函数自动退出
func Timeout(dur time.Duration, fn func(ctx context.Context)) error {
	return defaultProcess.goWithTimeout(dur, fn)
}
