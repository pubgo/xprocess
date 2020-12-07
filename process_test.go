package xprocess_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xprocess"
	"github.com/stretchr/testify/assert"
)

func TestCancel(t *testing.T) {
	fmt.Println(xprocess.Stack())
	cancel := xprocess.Go(func(ctx context.Context)  {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(time.Millisecond * 100)
				fmt.Println("g1")
			}
		}
	})

	time.Sleep(time.Second)
	fmt.Println(xprocess.Stack())
	cancel()
	time.Sleep(time.Second)
	fmt.Println(xprocess.Stack())
}

func TestName(t *testing.T) {
	fmt.Println(xprocess.Stack())
	for {
		xprocess.Go(func(ctx context.Context)  {
			time.Sleep(time.Second)
			fmt.Println("g2")
			return
		})

		xprocess.GoLoop(func(ctx context.Context)  {
			time.Sleep(time.Second)
			fmt.Println("g3")
			return
		})

		g := xprocess.NewGroup()
		g.Go(func(ctx context.Context)  {
			fmt.Println("g4")
		})

		g.Go(func(ctx context.Context) {
			fmt.Println("g5")
		})

		g.Go(func(ctx context.Context)  {
			fmt.Println("g6")
			xerror.Panic(xerror.Fmt("test error"))
		})

		g.Wait()

		g.Cancel()

		fmt.Println(xprocess.Stack())
		time.Sleep(time.Second)
	}
}

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
