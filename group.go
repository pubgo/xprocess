package xprocess

import (
	"context"
	"errors"
	"sync"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xlog"
)

type Group = group
type group struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewGroup
// 创建一个group对象, 可以带上默认的Context
func NewGroup(contexts ...context.Context) *group {
	var ctx = context.Background()
	if len(contexts) > 0 {
		ctx = contexts[0]
	}

	if ctx == nil {
		xerror.Exit(xerror.New("[ctx] should not be nil"))
	}

	_ctx, cancel := context.WithCancel(ctx)
	return &group{ctx: _ctx, cancel: cancel}
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
func (g *group) Go(fn func(ctx context.Context)) {
	if fn == nil {
		xerror.Next().Panic(errors.New("[fn] should not be nil"))
	}

	if err := g.ctx.Err(); err != nil {
		xerror.Next().Panic(err)
	}

	go func() {
		g.wg.Add(1)
		var counter = dataCounter(fn)
		defer func() {
			defer xerror.Resp(func(err xerror.XErr) { xlog.Error("group.Go handle error", xlog.Any("err", err)) })
			g.wg.Done()
			counter()
		}()
		fn(g.ctx)
	}()
}
