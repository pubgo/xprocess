package xprocess

import (
	"context"
	"github.com/pubgo/xerror"
	"go.uber.org/atomic"
	"reflect"

	"sync"
)

type group struct {
	ctx    context.Context
	cancel func()
	wg     sync.WaitGroup
	err    atomic.Error
}

func NewGroup(contexts ...context.Context) *group {
	var ctx = context.Background()
	if len(contexts) > 0 {
		ctx = contexts[0]
	}
	if ctx == nil {
		xerror.Exit(xerror.New("Context is nil"))
	}

	_ctx, cancel := context.WithCancel(ctx)
	return &group{ctx: _ctx, cancel: cancel}
}

func (g *group) Err() error {
	return g.err.Load()
}

func (g *group) Cancel() {
	g.cancel()
}

func (g *group) Wait() {
	g.wg.Wait()
	g.cancel()
}

func (g *group) Go(fn func(ctx context.Context) error) {
	if fn == nil {
		return
	}

	if g.ctx.Err() != nil {
		return
	}

	g.wg.Add(1)
	actual, _ := data.LoadOrStore(reflect.ValueOf(fn), atomic.NewInt32(0))
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
