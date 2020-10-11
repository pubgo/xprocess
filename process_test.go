package xprocess_test

import (
	"context"
	"fmt"
	"github.com/pubgo/xprocess"
	"testing"
	"time"
)

func TestName(t *testing.T) {
	fmt.Println(xprocess.Stack())
	for {
		xprocess.Go(func(ctx context.Context) error {
			time.Sleep(time.Second)

			return ctx.Err()
		})
		xprocess.Go(func(ctx context.Context) error {
			time.Sleep(time.Second)

			return ctx.Err()
		})
		xprocess.GoLoop(func(ctx context.Context) error {
			time.Sleep(time.Millisecond * 10)
			fmt.Println("ok")
			return ctx.Err()
		})

		g := xprocess.NewGroup(nil)
		g.Go(func(ctx context.Context) error {

			return nil
		})
		g.Wait()

		fmt.Println(xprocess.Stack())
	}
}
