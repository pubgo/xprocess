package xprocess_future

import (
	"fmt"
	"github.com/pubgo/xprocess/xprocess_abc"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAwaitFn(t *testing.T) {
	val := AwaitFn(http.Get, "https://www.cnblogs.com")
	assert.Nil(t, val.Err())
	assert.NotNil(t, val.Get())
	assert.Nil(t, val.Value(func(resp *http.Response, err error) {
		assert.Nil(t, err)
		assert.NotNil(t, resp)
	}))

	val = AwaitFn(func(i int) {}, 1)
	assert.Nil(t, val.Err())
	assert.Nil(t, val.Get())
	assert.Nil(t, val.Value(func() {}))
}

func TestAwait(t *testing.T) {
	val := AwaitFn(http.Get, "https://www.cnblogs.com")
	head := Await(val, func(resp *http.Response, err error) http.Header {
		assert.Nil(t, err)
		resp.Header.Add("aa", "11")
		return resp.Header
	})
	assert.Nil(t, head.Err())
	assert.NotNil(t, head.Get())
	assert.Equal(t, head.Get().(http.Header).Get("aa"), "11")
}

func promise1() xprocess_abc.IPromise {
	return Promise(func(g xprocess_abc.Future) {
		for i := 0; i < 10; i++ {
			val := AwaitFn(http.Get, "https://www.cnblogs.com")
			g.Yield(val)

			val = AwaitFn(http.Get, "https://www.cnblogs.com")
			val = Await(val, func(resp *http.Response, err error) (*http.Response, error) {
				resp.Header.Set("a", "b")
				return resp, err
			})
			g.YieldFn(val, func(resp *http.Response, err error) (*http.Response, error) {
				resp.Header.Set("b", "c")
				return resp, err
			})
		}
	})
}

func TestPromise(t *testing.T) {
	p := promise1()
	assert.Nil(t, p.RunUntilComplete())

	p = promise1()
	heads := p.Map(func(resp *http.Response, err error) http.Header { return resp.Header })
	for _,h := range heads.([]http.Header) {
		fmt.Println(h)
	}

	p.Cancelled()
}
