package xprocess

import (
	"fmt"
	"net/http"
	"testing"
)

func handleReq(i int) Value {
	fmt.Println("url", i)
	return Async(http.Get, "https://www.cnblogs.com")
}

func getData() IFuture {
	return Future(func(y Yield) {
		for i := 10; i > 0; i-- {
			i := i
			if i <= 3 {
				return
			}

			y.Return(handleReq(i))
		}
	}, 2)
}

func handleData() IFuture {
	return Future(func(y Yield) {
		getData().Await(func(data Value) {
			resp := data.Value().(*http.Response).Header
			y.Return(NewValue(resp, nil))
		})
	})
}

func TestStream(t *testing.T) {
	handleData().Await(func(data Value) {
		fmt.Println("dt", data.Value())
	})
}

func TestW1(t *testing.T) {
	val1 := Async(handleReq, 1)
	val2 := Async(handleReq, 1)
	val3 := Async(handleReq, 1)
	val4 := Async(handleReq, 1)

	fmt.Println(val1.Value(), val2.Value(), val3.Value(), val4.Value())
}
