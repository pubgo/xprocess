package xprocess

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"time"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xlog"
	"go.uber.org/atomic"
)

var ErrTimeout = xerror.New("timeout")
var Break = xerror.New("break")

var data sync.Map

func dataCounter(fn interface{}) func() {
	actual, _ := data.LoadOrStore(reflect.ValueOf(fn), atomic.NewInt32(0))
	actual.(*atomic.Int32).Inc()
	return func() { actual.(*atomic.Int32).Dec() }
}

type process struct{}

func (t *process) goCtx(fn func(ctx context.Context)) context.CancelFunc {
	if fn == nil {
		return func() { return }
	}

	ctx, cancel := context.WithCancel(context.Background())
	var counter = dataCounter(fn)
	go func() {
		defer func() {
			counter()
			cancel()
		}()

		defer xerror.Resp(func(err xerror.XErr) {
			xlog.Error("process.goCtx handle error", xlog.Any("err", err))
		})

		fn(ctx)
	}()

	return cancel
}

func (t *process) goLoopCtx(fn func(ctx context.Context) error) context.CancelFunc {
	if fn == nil {
		return func() { return }
	}

	ctx, cancel := context.WithCancel(context.Background())
	var counter = dataCounter(fn)
	go func() {
		defer func() {
			counter()
			cancel()
		}()

		defer xerror.Resp(func(err xerror.XErr) {
			xlog.Error("process.goLoopCtx handle error", xlog.Any("err", err))
		})

		for {
			select {
			case <-ctx.Done():
				return
			default:
				err := errors.Unwrap(fn(ctx))
				if err == Break {
					return
				}
				xerror.Panic(err)
			}
		}
	}()

	return cancel
}

func (t *process) goWithTimeout(dur time.Duration, fn func(ctx context.Context) error) error {
	if dur < 0 {
		return xerror.New("[dur] should not be less than zero")
	}

	if fn == nil {
		return xerror.New("[fn] should not be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var ch = make(chan error, 1)
	var counter = dataCounter(fn)
	go func() {
		defer func() {
			counter()
			cancel()
		}()

		defer xerror.Resp(func(err xerror.XErr) { ch <- err })

		ch <- fn(ctx)
	}()

	select {
	case err := <-ch:
		return xerror.Wrap(err)
	case <-time.After(dur):
		return ErrTimeout
	}
}

func (t *process) goWithDelay(dur time.Duration, fn func(ctx context.Context)) (context.CancelFunc, error) {
	if dur < 0 {
		return nil, xerror.New("[dur] should not be less than zero")
	}

	if fn == nil {
		return nil, xerror.New("[fn] should not be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())
	var counter = dataCounter(fn)
	go func() {
		defer func() {
			counter()
			cancel()
		}()

		defer xerror.Resp(func(err xerror.XErr) {
			dur = 0
			xlog.Error("process.goWithDelay handle error", xlog.Any("err", err))
		})

		fn(ctx)
	}()

	time.Sleep(dur)

	return cancel, nil
}
