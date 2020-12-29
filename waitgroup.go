package xprocess

import (
	"runtime"
	"sync"

	"go.uber.org/atomic"
)

type WaitGroup struct {
	wg    sync.WaitGroup
	count atomic.Int32
	Num   int32
}

func (t *WaitGroup) check() {
	if t.Num == 0 {
		t.Num = int32(runtime.NumCPU() * 2)
	}

	if t.count.Load() > t.Num {
		t.wg.Wait()
	}
}

func (t *WaitGroup) Inc()          { t.count.Inc(); t.check(); t.wg.Add(1) }
func (t *WaitGroup) Add(delta int) { t.count.Add(int32(delta)); t.check(); t.wg.Add(delta) }
func (t *WaitGroup) Done()         { t.count.Dec(); t.wg.Done() }
func (t *WaitGroup) Wait()         { t.wg.Wait() }
