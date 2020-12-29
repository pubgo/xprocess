package xprocess

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/pubgo/xerror"
)

func handleReq(i int) (resp *http.Response, err error) {
	fmt.Println("url", i)
	return http.Get("https://www.cnblogs.com")
}

func asyncHandleReq(i int) FutureValue {
	fmt.Println("url", i)
	return Async(http.Get, "https://www.cnblogs.com")
}

func getData() Future {
	return Promise(func(y Yield) {
		defer xerror.RespExit()

		for i := 10; i > 0; i-- {
			i := i
			if i <= 3 {
				return
			}

			val := Async(http.Get, "https://www.cnblogs.com")
			head := Await(val, func(resp *http.Response, err error) http.Header {
				xerror.Panic(err)
				resp.Header.Add("test", "11111")
				return resp.Header
			})

			val1 := Await(asyncHandleReq(1), func(resp *http.Response, err error) *http.Response{
				resp.Header.Set("dddd", "hhhh")
				head.Value(func(head http.Header) {resp.Header = head})
				return resp
			})

			y.Yield(func() {
				resp, err := http.Get("https://www.cnblogs.com")
				xerror.Panic(err)
				resp.Header.Add("testsssss", "11111")
				y.Yield(resp)
			})
		}
	})
}

func handleData() Future {
	return Promise(func(y Yield) {
		s := getData()
		s.Value(func(resp *http.Response) {
			head := resp.Header
			head.Add("test1111", "22222")
			y.Yield(head)
		})
	})
}

func TestStream(t *testing.T) {
	s := handleData().
		Err(func(err error) { fmt.Println(err) }).
		Value(func(head http.Header) {
			fmt.Println("dt", head)
		})

	for val := range s.Chan() {
		fmt.Println(val)
	}
}

func TestAsync(t *testing.T) {
	val1 := Async(handleReq, 1)
	val2 := Async(handleReq, 2)
	val3 := Async(handleReq, 3)
	val4 := Async(handleReq, 4)

	fmt.Printf("%#v, %#v, %#v, %#v\n", val1.Get(), val2.Get(), val3.Get(), val4.Get())
}

func TestGetData(t *testing.T) {
	getData().Value(func(resp *http.Response, err error) {
		fmt.Println(resp)
	})
}

func handleData2() Future {
	return Promise(func(y Yield) {
		defer xerror.Resp(func(err xerror.XErr) {
			y.Cancel()
		})

		for i := 10; i > 0; i-- {

			i := i
			y.Yield(i)
			y.Yield(func() {
				if i == 5 {
					//xerror.Panic(xerror.New("error test"))
				}

				resp, err := http.Get("https://www.cnblogs.com")
				xerror.Panic(err)

				y.Yield(resp)
			})
		}
	})
}

func TestName11w(t *testing.T) {
	s := handleData2().
		Err(func(err error) {
			fmt.Println(err)
		}).
		Value(func(i interface{}) {
			fmt.Println(i)
		})
	s.Done()
}
