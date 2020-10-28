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
}

func TestName(t *testing.T) {
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
		xerror.Exit(g.Go(func(ctx context.Context) error {
			fmt.Println("g4")
			return nil
		}))

		xerror.Exit(g.Go(func(ctx context.Context) error {
			fmt.Println("g5")
			return nil
		}))

		xerror.Exit(g.Go(func(ctx context.Context) error {
			fmt.Println("g6")
			return xerror.Fmt("test error")
		}))

		g.Wait()
		fmt.Println(g.Err())

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
