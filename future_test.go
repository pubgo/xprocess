package xprocess

import (
	"fmt"
	"github.com/pubgo/xtest"
	"net/http"
	"testing"
	"time"
)

func handleReq(i int) Value {
	fmt.Println("url", i)
	return Async(http.Get, "https://www.cnblogs.com")
}

//func handleReq1(i int) (*http.Response, error) {
//	fmt.Println("url", i)
//	return Async(time.Sleep, time.Duration(xtest.Range(0, i))*time.Millisecond*100)
//}

func getDataHeader() IFuture {
	return Future(func(y Yield) {
		for i := 10; i > 0; i-- {
			i := i
			if i <= 3 {
				return
			}

			y.Await(handleReq(i), func(resp *http.Response, err error) {
				y.Yield(resp.Header)
			})
			//y.Yield(handleReq(i))
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

			//y.Await(handleReq(i), func(resp *http.Response, err error) *http.Response {
			//	xerror.Panic(err)
			//
			//	return resp
			//})

			y.Yield(Async(http.Get, "https://www.cnblogs.com"))
		}
	}, 2)
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

	select {}
}

func TestAsync(t *testing.T) {
	val1 := Async(time.Sleep, time.Duration(xtest.Range(0, 10))*time.Millisecond*100)
	val2 := Async(time.Sleep, time.Duration(xtest.Range(0, 10))*time.Millisecond*100)
	val3 := Async(time.Sleep, time.Duration(xtest.Range(0, 10))*time.Millisecond*100)
	val4 := Async(time.Sleep, time.Duration(xtest.Range(0, 10))*time.Millisecond*100)

	fmt.Printf("%#v, %#v, %#v, %#v\n", val1.Get(), val2.Get(), val3.Get(), val4.Get())
}

func TestGetData(t *testing.T) {
	getData().Value(func(resp *http.Response) {
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
