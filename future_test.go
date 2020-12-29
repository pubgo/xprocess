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

			y.Await(handleReq(i), func(resp *http.Response, err error) (*http.Response, error) {
				xerror.Panic(err)

				resp.Header.Add("test", "11111")
				return resp, err
			})

			y.Yield(Async(http.Get, "https://www.cnblogs.com"))
		}
	})
}

func handleData() IFuture {
	return Future(func(y Yield) {
		getData().Value(func(resp *http.Response, err error) {
			y.Yield(resp.Header)
		})
	})
}

func TestStream(t *testing.T) {
	handleData().Value(func(head http.Header) {
		fmt.Println("dt", head)
	})
}

func TestAsync(t *testing.T) {
	val1 := handleReq(1)
	val2 := handleReq(2)
	val3 := handleReq(3)
	val4 := handleReq(4)

	fmt.Printf("%#v, %#v, %#v, %#v\n", val1.Get(), val2.Get(), val3.Get(), val4.Get())
}

func TestGetData(t *testing.T) {
	getData().Value(func(resp *http.Response, err error) {
		fmt.Println(resp)
	})
}

func handleData2() IFuture {
	return Future(func(y Yield) {
		for i := 10; i > 0; i-- {
			y.Yield(i)
		}
	})
}

func TestName11w(t *testing.T) {
	handleData2().Value(func(i int) {
		fmt.Println(i)
	})
}
