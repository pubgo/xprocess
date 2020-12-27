package xprocess

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/pubgo/xerror"
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

func getDataWithAwait() IFuture {
	return Future(func(y Yield) {
		defer xerror.RespExit()

		for i := 10; i > 0; i-- {
			if i <= 3 {
				return
			}

			fmt.Println(y.Yield(handleReq, i))
		}
	}, 2)
}

func handleData() IFuture {
	return Future(func(y Yield) {
		getData().Await(func(data interface{}) {
			y.Return(data.(*http.Response).Header)
		})
	})
}

func handleData1() IFuture {
	return Future(func(y Yield) {
		getDataWithAwait().Await(func(data interface{}) {
			y.Return(data.(*http.Response).Header)
		})
	})
}

func TestStream(t *testing.T) {
	handleData().Await(func(data interface{}) {
		fmt.Println("dt", data)
	})
}

func TestStream1(t *testing.T) {
	handleData1().Await(func(data interface{}) {
		fmt.Println("dt", data)
	})
}

func TestW1(t *testing.T) {
	val1 := Async(handleReq, 1)
	val2 := Async(handleReq, 1)
	val3 := Async(handleReq, 1)
	val4 := Async(handleReq, 1)

	fmt.Println(val1.Value(), val2.Value(), val3.Value(), val4.Value())
}
