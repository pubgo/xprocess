package xprocess

import (
	"runtime"
	"sync"
	"sync/atomic"
	_ "unsafe"
)

//go:linkname state sync.(*WaitGroup).state
func state(*sync.WaitGroup) (*uint64, *uint32)

type WaitGroup struct {
	_          int8
	Check      int8
	Concurrent int16
	sync.WaitGroup
}

func (t *WaitGroup) Count() int16 {
	count, _ := state(&t.WaitGroup)
	return int16(atomic.LoadUint64(count) >> 32)
}

func (t *WaitGroup) check() {
	if t.Check == 0 {
		return
	}

	if t.Concurrent == 0 {
		t.Concurrent = int16(runtime.NumCPU() * 2)
	}

	if t.Count() >= t.Concurrent {
		t.WaitGroup.Wait()
	}
}

func (t *WaitGroup) Inc()          { t.check(); t.WaitGroup.Add(1) }
func (t *WaitGroup) Dec()          { t.check(); t.WaitGroup.Done() }
func (t *WaitGroup) Add(delta int) { t.check(); t.WaitGroup.Add(delta) }
