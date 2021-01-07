package xprocess_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/pubgo/xerror"
	"github.com/pubgo/xprocess"
	"github.com/pubgo/xprocess/xprocess_group"
	"github.com/stretchr/testify/assert"
)

func TestCancel(t *testing.T) {
	cancel := xprocess.Go(func(ctx context.Context) {
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
	cancel()
	time.Sleep(time.Second)
}

func TestName(t *testing.T) {
	for {
		xprocess.Go(func(ctx context.Context) {
			time.Sleep(time.Second)
			fmt.Println("g2")
			return
		})

		xprocess.GoLoop(func(ctx context.Context) {
			time.Sleep(time.Second)
			fmt.Println("g3")
		})

		g := xprocess_group.New()
		g.Go(func(ctx context.Context) {
			fmt.Println("g4")
		})

		g.Go(func(ctx context.Context) {
			fmt.Println("g5")
		})

		g.Go(func(ctx context.Context) {
			fmt.Println("g6")
			xerror.Panic(xerror.Fmt("test error"))
		})

		g.Wait()

		g.Cancel()

		time.Sleep(time.Second)
	}
}

func TestTimeout(t *testing.T) {
	err := xprocess.Timeout(time.Second, func(ctx context.Context) {
		time.Sleep(time.Millisecond * 990)
	})
	assert.Nil(t, err)

	err = xprocess.Timeout(time.Second, func(ctx context.Context) {
		time.Sleep(time.Second + time.Millisecond*10)
	})
	assert.NotNil(t, err)
}
