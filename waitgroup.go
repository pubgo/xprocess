package xprocess

import "github.com/pubgo/xprocess/xprocess_waitgroup"

type WaitGroup = xprocess_waitgroup.WaitGroup

func NewWaitGroup(check uint8, c ...uint16) WaitGroup {
	return xprocess_waitgroup.New(check, c...)
}
