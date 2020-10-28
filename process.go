package xprocess

import (
	"context"
	"github.com/pubgo/xerror"
	"go.uber.org/atomic"
	"reflect"
	"sync"
	"time"
)

var data sync.Map

func dataCounter(fn interface{}) func() {
	actual, _ := data.LoadOrStore(reflect.ValueOf(fn), atomic.NewInt32(0))
	actual.(*atomic.Int32).Inc()
	return func() {
		actual.(*atomic.Int32).Dec()
	}
}

type process struct{}

func (t *process) goCtx(fn func(ctx context.Context) error) func() error {
	if fn == nil {
		return func() error {
			return nil
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	var err error
	var counter = dataCounter(fn)
	go func() {
		defer func() {
			xerror.RespErr(&err)
			counter()
			cancel()
		}()

		err = fn(ctx)
	}()

	return func() error {
		cancel()
		return err
	}
}

func (t *process) goLoopCtx(fn func(ctx context.Context) error) func() error {
	if fn == nil {
		return func() error {
			return nil
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	var err error
	var counter = dataCounter(fn)
	go func() {
		defer func() {
			xerror.RespErr(&err)
			counter()
			cancel()
		}()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				if _err := fn(ctx); _err != nil {
					err = _err
					return
				}
			}
		}
	}()

	return func() error {
		cancel()
		return err
	}
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
			xerror.Resp(func(err xerror.XErr) {
				ch <- err
			})
			counter()
			cancel()
		}()

		ch <- fn(ctx)
	}()

	select {
	case err := <-ch:
		return err
	case <-time.After(dur):
		return xerror.New("timeout")
	}
}
