package xprocess

import (
	"fmt"
	"github.com/pubgo/xerror"
	"net/http"
	"testing"
)

func handleReq(i int) (*http.Response, error) {
	fmt.Println("url", i)
	return http.Get("https://www.cnblogs.com")
}

func getData() IFuture {
	return Future(func(y Yield) {
		for i := 10; i > 0; i-- {
			i := i
			if i <= 3 {
				return
			}

			y.Go(func() {
				resp, err := handleReq(i)
				if err != nil {
					panic(err)
				}

				y.Return(resp)
			})
		}
	}, 2)
}

func getDataWithAwait() IFuture {
	return Future(func(y Yield) {
		for i := 10; i > 0; i-- {
			i1 := i
			if i1 <= 3 {
				return
			}

			xerror.Panic(y.Yield(handleReq, i1))
		}
	}, 2)
}

func handleData() IFuture {
	s := getData()
	return s.Future(func(y Yield) {
		for resp := range s.Chan() {
			y.Return(resp.(*http.Response).Header)
		}
	})
}

func handleData1() IFuture {
	s := getDataWithAwait()
	return s.Future(func(y Yield) {
		for resp := range s.Chan() {
			y.Return(resp.(*http.Response).Header)
		}
	})
}

func TestStream(t *testing.T) {
	s := handleData()
	go s.Await(func(data interface{}) {
		fmt.Println("dt", data)
	})

	for dt := range s.Chan() {
		fmt.Println("data", dt)
	}
}

func TestStream1(t *testing.T) {
	s := handleData1()
	go s.Await(func(data interface{}) {
		fmt.Println("dt", data)
	})

	for dt := range s.Chan() {
		fmt.Println("data", dt)
	}
}
