package xprocess

import (
	"runtime"
	"sync"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xlog"
	"go.uber.org/atomic"
)

type Stream interface {
	Stream(fn func(y Yield), nums ...int) Stream
	Await(func(data interface{}))
	Chan() chan interface{}
}

type Yield interface {
	Go(fn func())
	Yield(data interface{})
}

type stream struct {
	wg    sync.WaitGroup
	num   int32
	count atomic.Int32
	data  chan interface{}
}

func (s *stream) Stream(fn func(y Yield), nums ...int) Stream {
	num := runtime.NumCPU() * 2
	if len(nums) > 0 {
		num = nums[0]
	}

	stm := &stream{num: int32(num), data: make(chan interface{})}
	go fn(stm)
	return stm
}

func (s *stream) Chan() chan interface{} {
	return s.data
}

func (s *stream) Yield(data interface{}) {
	s.data <- data
}

func (s *stream) Await(fn func(data interface{})) {
	go func() {
		for data := range s.data {
			fn(data)
		}
	}()
}

func (s *stream) Go(fn func()) {
	if s.count.Load() >= s.num {
		s.wg.Wait()
	}

	s.count.Inc()
	s.wg.Add(1)

	go func() {
		defer func() {
			xerror.Resp(func(err xerror.XErr) { xlog.Error("stream.Go handle error", xlog.Any("err", err)) })
			s.count.Dec()
			s.wg.Done()
		}()

		fn()
	}()
}

func NewStream(fn func(y Yield), nums ...int) Stream {
	num := runtime.NumCPU() * 2
	if len(nums) > 0 {
		num = nums[0]
	}

	s := &stream{num: int32(num), data: make(chan interface{})}
	go fn(s)
	return s
}
