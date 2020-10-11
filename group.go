package xprocess

import (
	"context"
	"github.com/pubgo/xerror"
	"github.com/pubgo/xerror/xerror_util"
	"go.uber.org/atomic"

	"sync"
)

type group struct {
	ctx    context.Context
	Cancel func()
	wg     sync.WaitGroup
	err    atomic.Error
}

func NewGroup(ctx context.Context) *group {
	if ctx == nil {
		ctx = context.Background()
	}

	_ctx, cancel := context.WithCancel(ctx)
	return &group{ctx: _ctx, Cancel: cancel}
}

func (g *group) Err() error {
	return g.err.Load()
}

func (g *group) Wait() {
	g.wg.Wait()
	g.Cancel()
}

func (g *group) Go(fn func(ctx context.Context) error) {
	if fn == nil {
		return
	}

	if g.ctx.Err() != nil {
		return
	}

	g.wg.Add(1)
	actual, _ := data.LoadOrStore(xerror_util.CallerWithFunc(fn), atomic.NewInt32(0))
	actual.(*atomic.Int32).Inc()

	go func() {
		defer xerror.Resp(func(err xerror.XErr) {
			g.err.Store(err)
		})
		defer g.wg.Done()
		defer actual.(*atomic.Int32).Dec()

		if err := fn(g.ctx); err != nil {
			g.err.Store(err)
		}
	}()
}
