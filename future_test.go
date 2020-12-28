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

func handleReq1(i int) (*http.Response, error) {
	fmt.Println("url", i)
	return http.Get("https://www.cnblogs.com")
}

func getDataHeader() IFuture {
	return Future(func(y Yield) {
		for i := 10; i > 0; i-- {
			i := i
			if i <= 3 {
				return
			}

			Await1(handleReq(i), func(resp *http.Response, err error) {
				y.Yield(resp.Header)
			})

			//y.Yield(handleReq(i))
			Await(http.Get, "https://www.cnblogs.com")(func(resp *http.Response, err error) {
				y.Yield(resp.Header)
			})
		}
	}, 2)
}

func getData() IFuture {
	return Future(func(y Yield) {
		for i := 10; i > 0; i-- {
			i := i
			if i <= 3 {
				return
			}

			//y.Yield(handleReq(i))
			//Await(http.Get, "https://www.cnblogs.com")(func() {
			//	y.Yield()
			//})
		}
	}, 2)
}

func handleData() IFuture {
	return Future(func(y Yield) {
		getData().Value(func(resp *http.Response) {
			y.Yield(resp.Header)
		})
	})
}

func TestStream(t *testing.T) {
	handleData().Value(func(head http.Header) {
		fmt.Println("dt", head)
	})
}

func TestStream1(t *testing.T) {
	for dt := range handleData().Chan() {
		fmt.Println("dt", dt)
	}
}

func TestW1(t *testing.T) {
	val1 := Async(handleReq, 1)
	val2 := Async(handleReq, 1)
	val3 := Async(handleReq, 1)
	val4 := Async(handleReq, 1)

	fmt.Printf("%#v, %#v, %#v, %#v\n", val1.Get(), val2.Get(), val3.Get(), val4.Get())
}
