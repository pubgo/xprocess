package xprocess

import (
	"context"
	"errors"
	"time"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xlog"
	"github.com/pubgo/xprocess/xprocess_errs"
)

func Break() { panic(xprocess_errs.ErrBreak) }

type process struct{}

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
			if errors.Unwrap(err) == xprocess_errs.ErrBreak {
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

func (t *process) goWithTimeout(dur time.Duration, fn func(ctx context.Context)) (err error) {
	defer xerror.RespErr(&err)

	xerror.Assert(dur < 0, "[dur] should not be less than zero")
	xerror.Assert(fn == nil, "[fn] should not be nil")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var ch = make(chan error, 1)
	go func() {
		defer cancel()
		defer xerror.Resp(func(err xerror.XErr) { ch <- err })

		fn(ctx)
		ch <- nil
	}()

	select {
	case err := <-ch:
		return xerror.Wrap(err)
	case <-time.After(dur):
		return xprocess_errs.ErrTimeout
	}
}

func (t *process) goWithDelay(dur time.Duration, fn func(ctx context.Context)) context.CancelFunc {
	xerror.Assert(dur < 0, "[dur] should not be less than zero")
	xerror.Assert(fn == nil, "[fn] should not be nil")

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		defer xerror.Resp(func(err xerror.XErr) {
			dur = 0
			xlog.Error("process.goWithDelay error", xlog.Any("err", err))
		})

		fn(ctx)
	}()

	if dur != 0 {
		time.Sleep(dur)
	}

	return cancel
}
