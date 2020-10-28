package xprocess

import (
	"context"
	"sync"

	"github.com/pubgo/xerror"
	"go.uber.org/atomic"
)

type Group = group
type group struct {
	ctx    context.Context
	cancel func()
	wg     sync.WaitGroup
	err    atomic.Error
}

// NewGroup
// 创建一个group对象, 可以带上默认的Context
func NewGroup(contexts ...context.Context) *group {
	var ctx = context.Background()
	if len(contexts) > 0 {
		ctx = contexts[0]
	}

	if ctx == nil {
		xerror.Exit(xerror.New("[ctx] is nil"))
	}

	_ctx, cancel := context.WithCancel(ctx)
	return &group{ctx: _ctx, cancel: cancel}
}

// Err
// 返回func执行时遇到的错误
func (g *group) Err() error {
	return g.err.Load()
}

// Cancel
// 停止正在运行的函数
func (g *group) Cancel() {
	g.cancel()
}

// Wait
// 等待正在运行的函数
func (g *group) Wait() {
	g.wg.Wait()
	g.cancel()
}

// Go
// 运行一个goroutine
func (g *group) Go(fn func(ctx context.Context) error) error {
	if fn == nil {
		return xerror.New("[fn] should not be nil")
	}

	if err := g.ctx.Err(); err != nil {
		return xerror.Wrap(err)
	}

	g.wg.Add(1)
	var counter = dataCounter(fn)
	go func() {
		defer func() {
			xerror.Resp(func(err xerror.XErr) {
				g.err.Store(err)
			})
			g.wg.Done()
			counter()
		}()

		g.err.Store(fn(g.ctx))
	}()
	return nil
}
