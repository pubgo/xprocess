package xprocess_waitgroup

import (
	"runtime"
	"sync"
	"sync/atomic"
	_ "unsafe"
)

//go:linkname state sync.(*WaitGroup).state
func state(*sync.WaitGroup) (*uint64, *uint32)

func New(check uint8, c ...uint16) WaitGroup {
	cc := uint16(runtime.NumCPU() * 2)
	if len(c) > 0 {
		cc = c[0]
	}

	return WaitGroup{Check: check, Concurrent: cc}
}

type WaitGroup struct {
	_          int8
	Check      uint8
	Concurrent uint16
	wg         sync.WaitGroup
}

func (t *WaitGroup) Count() uint16 {
	count, _ := state(&t.wg)
	return uint16(atomic.LoadUint64(count) >> 32)
}

func (t *WaitGroup) check() {
	if t.Check == 0 {
		return
	}

	if t.Concurrent == 0 {
		t.Concurrent = uint16(runtime.NumCPU() * 2)
	}

	if t.Count() >= t.Concurrent {
		t.wg.Wait()
	}
}

func (t *WaitGroup) EnableCheck()  { t.Check = 1 }
func (t *WaitGroup) Inc()          { t.check(); t.wg.Add(1) }
func (t *WaitGroup) Dec()          { t.wg.Done() }
func (t *WaitGroup) Done()         { t.wg.Done() }
func (t *WaitGroup) Wait()         { t.wg.Wait() }
func (t *WaitGroup) Add(delta int) { t.check(); t.wg.Add(delta) }
