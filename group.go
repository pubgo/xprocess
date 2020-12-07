package xprocess

import (
	"context"
	"errors"
	"runtime"
	"sync"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xlog"
	"go.uber.org/atomic"
)

type Group = group
type group struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	c      uint32
	n      atomic.Uint32
}

type GroupOption func(*group)

// NewGroup
// 创建一个group对象, 可以带上默认的Context
func NewGroup(opts ...GroupOption) *group {
	_ctx, cancel := context.WithCancel(context.Background())
	g := &group{ctx: _ctx, cancel: cancel, c: uint32(runtime.NumCPU() * 2)}

	for i := range opts {
		opts[i](g)
	}
	return g
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

	g.n.Inc()
	g.wg.Add(1)
	var counter = dataCounter(fn)

	select {
	case <-g.ctx.Done():
		return
	default:
		go func() {
			defer func() {
				defer xerror.Resp(func(err xerror.XErr) { xlog.Error("group.Go handle error", xlog.Any("err", err)) })
				g.n.Dec()
				g.wg.Done()
				counter()
			}()
			fn(g.ctx)
		}()

		if g.n.Load() > g.c {
			g.wg.Wait()
		}
	}
}

func WithConcurrency(c uint32) GroupOption {
	if c < 0 {
		c = uint32(runtime.NumCPU() * 2)
	}

	return func(g *group) {
		g.c = c
	}
}
