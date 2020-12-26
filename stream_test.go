package xprocess

import (
	"fmt"
	"net/http"
	"testing"
)

func getData() Stream {
	return NewStream(func(y Yield) {
		for i := 100; i > 0; i-- {
			y.Go(func() {
				resp, err := http.Get("https://www.cnblogs.com")
				if err != nil {
					y.Yield(err)
				}
				y.Yield(resp)
			})
		}
	})
}

func handleData() Stream {
	s := getData()
	return s.Stream(func(y Yield) {
		for resp := range s.Chan() {
			y.Yield(resp.(*http.Response).Header)
		}
	})
}

func TestStream(t *testing.T) {
	s := handleData()
	s.Await(func(data interface{}) {
		fmt.Println("data", data)
	})

	for dt := range s.Chan() {
		fmt.Println("dt", dt)
	}
}
