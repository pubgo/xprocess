package xprocess_future

import (
	"github.com/pubgo/xprocess/xprocess_abc"
)

var _ xprocess_abc.Future = (*future)(nil)

type future struct{ p *promise }

func (f *future) Yield(data interface{})                               { f.p.yield(data) }
func (f *future) YieldFn(val xprocess_abc.FutureValue, fn interface{}) { f.p.await(val, fn) }
