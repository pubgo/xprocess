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

func getData() IPromise {
	return Promise(func(y Future) {
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

			y.Yield(asyncHandleReq(1), func(resp *http.Response, err error) *http.Response {
				resp.Header.Set("dddd", "hhhh")
				head.Value(func(head http.Header) { resp.Header = head })
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

func handleData() IPromise {
	return Promise(func(y Future) {
		s := getData()
		s.Value(func(resp *http.Response) {
			head := resp.Header
			head.Add("test1111", "22222")
			y.Yield(head)
		})

	})
}

func TestAsync(t *testing.T) {
	val1 := asyncHandleReq(1)
	val2 := asyncHandleReq(2)
	val3 := asyncHandleReq(3)
	val4 := Async(func(i int) {
		panic("sss")
	}, 1, 2)

	head := Await(val1, func(resp *http.Response, err error) http.Header {
		xerror.Panic(err)
		val2.Value(func(resp1 *http.Response, err1 error) {
			resp1.Header.Set("test", "11111")
			resp.Header = resp1.Header
		})
		return resp.Header
	})

	fmt.Printf("%s\n, %s\n, %s\n, %s\n, %s\n", val1, val2, val3, val4, head)
}

func TestGetData(t *testing.T) {
	defer xerror.RespExit()
	getData().Value(func(resp *http.Response) {
		fmt.Println(resp)
	})
}

func handleData2() IPromise {
	return Promise(func(g Future) {
		for i := 10; i > 0; i-- {
			i := i
			g.Yield(i)
			g.Yield(func() {
				if i == 5 {
					xerror.Panic(xerror.New("error test"))
				}

				resp, err := http.Get("https://www.cnblogs.com")
				xerror.Panic(err)
				g.Yield(resp)

				asyncHandleReq(1).Value(func(resp *http.Response, _ error) {
					g.Yield(resp)
				})

				g.Await(asyncHandleReq(1), func(resp *http.Response, _ error) {
					g.Yield(resp)
				})
			})
		}
	})
}

func TestName11w(t *testing.T) {
	s := handleData2()
	s.Value(func(i interface{}) {
		fmt.Println(i)
	})
	fmt.Printf("%+v\n", s.Wait())
}

func TestMap(t *testing.T) {
	dt := Map([]string{"111", "222"}, func(s string) string { return s + "mmm" })
	fmt.Println(dt.([]string)[0])
}
