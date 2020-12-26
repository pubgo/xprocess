package xprocess

import (
	"go.uber.org/atomic"
	"net/http"
)

type Stream interface {
	Await() chan interface{}
	Go(fn func())
	Yield(data interface{})
}

type stream struct {
	data   chan interface{}
	cancel atomic.Bool
}

func (s stream) Yield(data interface{}) {
	s.data <- data
}

func (s stream) Await() chan interface{} {
	return s.data
}

func (s stream) Go(fn func()) {
	go fn()
}

func NewStream() Stream {
	var s = &stream{data: make(chan interface{})}
	return s
}

func getData() Stream {
	s := NewStream()
	for i := 10; i > 0; i-- {
		s.Go(func() {
			resp, err := http.Get("https://www.oschina.net/news/124525/tokio-1-0-released")
			if err != nil {
				s.Yield(err)
			}
			s.Yield(resp)
		})
	}
	return s
}
