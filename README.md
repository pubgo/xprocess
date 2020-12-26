# xprocess
> 一个通用的机制去管理goroutine的生命周期

1. 通过xprocess.Go替换原生的直接启动goroutine的方式
2. 通过context传递, 管理goroutine的生命周期
3. GoLoop是对goroutine中有for的封装和替换
4. Group是一个WaitGroup的wrap,不过增加了goroutine统计和可退出机制
5. Stack会得到内存中所有在运行的goroutine, 以及数量

## Examples

```go
fmt.Println(xprocess.Stack())
cancel := xprocess.Go(func(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            time.Sleep(time.Millisecond * 100)
            fmt.Println("g1")
        }
    }
})

time.Sleep(time.Second)
fmt.Println(xprocess.Stack())
fmt.Println(cancel())
time.Sleep(time.Second)
fmt.Println(xprocess.Stack())
```

```go
fmt.Println(xprocess.Stack())
for {
    xprocess.Go(func(ctx context.Context) error {
        time.Sleep(time.Second)
        fmt.Println("g2")
        return ctx.Err()
    })
    xprocess.GoLoop(func(ctx context.Context) error {
        time.Sleep(time.Second)
        fmt.Println("g3")
        return ctx.Err()
    })

    g := xprocess.NewGroup()
    g.Go(func(ctx context.Context) error {
        fmt.Println("g4")
        return nil
    })
    g.Go(func(ctx context.Context) error {
        fmt.Println("g5")
        return nil
    })
    g.Go(func(ctx context.Context) error {
        fmt.Println("g6")
        return xerror.Fmt("test error")
    })
    g.Wait()
    fmt.Println(g.Err())

    g.Cancel()

    fmt.Println(xprocess.Stack())
    time.Sleep(time.Second)
}
```

### Timeout
```go
func TestTimeout(t *testing.T) {
	err := xprocess.Timeout(time.Second, func(ctx context.Context) error {
		time.Sleep(time.Millisecond * 990)
		return nil
	})
	assert.Nil(t, err)

	err = xprocess.Timeout(time.Second, func(ctx context.Context) error {
		time.Sleep(time.Second + time.Millisecond*10)
		return nil
	})
	assert.NotNil(t, err)
}
```

### Stream

```go
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
```
