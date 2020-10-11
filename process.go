package xprocess

import (
	"context"
	"github.com/pubgo/xerror"
	"github.com/pubgo/xerror/xerror_util"
	"go.uber.org/atomic"
	"sync"
)

var data sync.Map

type process struct {
}

func (t *process) goCtx(fn func(ctx context.Context) error) func() error {
	if fn == nil {
		return func() error {
			return nil
		}
	}

	actual, _ := data.LoadOrStore(xerror_util.CallerWithFunc(fn), atomic.NewInt32(0))
	actual.(*atomic.Int32).Inc()

	ctx, cancel := context.WithCancel(context.Background())
	var err error
	go func() {
		defer xerror.RespErr(&err)
		defer cancel()
		defer actual.(*atomic.Int32).Dec()
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

	actual, _ := data.LoadOrStore(xerror_util.CallerWithFunc(fn), atomic.NewInt32(0))
	actual.(*atomic.Int32).Inc()

	ctx, cancel := context.WithCancel(context.Background())
	var err error
	go func() {
		defer xerror.RespErr(&err)
		defer cancel()
		defer actual.(*atomic.Int32).Dec()

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
