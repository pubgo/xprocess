package xprocess

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xlog"
	"github.com/pubgo/xprocess/xprocess_errs"
)

func Break() { panic(xprocess_errs.ErrBreak) }

type process struct{}

func (t *process) memStatsPrint() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("HeapAlloc = %v HeapIdel= %v HeapSys = %v  HeapReleased = %v\n", m.HeapAlloc/1024, m.HeapIdle/1024, m.HeapSys/1024, m.HeapReleased/1024)
}

func (t *process) costWith(fn func()) (dur time.Duration) {
	xerror.Assert(fn == nil, "[fn] should not be nil")

	defer func(start time.Time) { dur = time.Since(start) }(time.Now())

	fn()
	return
}

func (t *process) count(n int) <-chan int {
	var c = make(chan int)
	go func() {
		defer close(c)
		for i := 0; i < n; i++ {
			c <- i
		}
	}()
	return c
}

// tick 简单定时器
// Example: tick(100, time.Second)
func (t *process) tick(args ...interface{}) <-chan time.Time {
	var n int
	var dur time.Duration

	for _, arg := range args {
		xerror.Assert(arg == nil, "[arg] should not be nil")

		switch ag := arg.(type) {
		case int:
			n = ag
		case time.Duration:
			dur = ag
		}
	}

	if n <= 0 {
		n = 1
	}

	if dur <= 0 {
		dur = time.Second
	}

	var c = make(chan time.Time)
	go func() {
		defer close(c)

		tk := time.NewTicker(dur)
		for i := 0; ; i++ {
			if i == n {
				tk.Stop()
				break
			}

			c <- <-tk.C
		}
	}()

	return c
}

func (t *process) try(fn func()) (err error) {
	defer xerror.RespErr(&err)
	fn()
	return
}

func (t *process) goCtx(fn func(ctx context.Context)) context.CancelFunc {
	xerror.Assert(fn == nil, "[fn] should not be nil")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		defer xerror.Resp(func(err xerror.XErr) { xlog.Error("process.goCtx error", xlog.Any("err", err)) })

		fn(ctx)
	}()

	return cancel
}

func (t *process) goLoopCtx(fn func(ctx context.Context)) context.CancelFunc {
	xerror.Assert(fn == nil, "[fn] should not be nil")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		defer xerror.Resp(func(err xerror.XErr) {
			if xerror.Unwrap(err) == xprocess_errs.ErrBreak {
				return
			}

			xlog.Error("process.goLoopCtx error", xlog.Any("err", err))
		})

		for {
			select {
			case <-ctx.Done():
				return
			default:
				fn(ctx)
			}
		}
	}()

	return cancel
}

func (t *process) goWithTimeout(dur time.Duration, fn func(ctx context.Context)) (gErr error) {
	defer xerror.RespErr(&gErr)

	xerror.Assert(dur <= 0, "[dur] should not be less than zero")
	xerror.Assert(fn == nil, "[fn] should not be nil")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var ch = make(chan struct{})
	go func() {
		defer close(ch)
		defer xerror.RespErr(&gErr)

		fn(ctx)
	}()

	select {
	case <-ch:
		return
	case <-time.After(dur):
		return xprocess_errs.ErrTimeout
	}
}

func (t *process) goWithDelay(dur time.Duration, fn func()) (gErr error) {
	defer xerror.RespErr(&gErr)

	xerror.Assert(dur <= 0, "[dur] should not be less than zero")
	xerror.Assert(fn == nil, "[fn] should not be nil")

	go func() {
		defer xerror.Resp(func(err xerror.XErr) {
			dur = 0
			gErr = err.WrapF("process.goWithDelay error")
		})

		fn()
	}()

	if dur != 0 {
		time.Sleep(dur)
	}

	return
}
