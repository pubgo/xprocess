package xprocess

import (
	"context"
	"encoding/json"
	"go.uber.org/atomic"
)

var defaultProcess = &process{}

func Go(fn func(ctx context.Context) error) func() error {
	return defaultProcess.goCtx(fn)
}

func GoLoop(fn func(ctx context.Context) error) func() error {
	return defaultProcess.goLoopCtx(fn)
}

func Stack() string {
	var _data = make(map[string]int32)
	data.Range(func(key, value interface{}) bool {
		_data[key.(string)] = value.(*atomic.Int32).Load()
		return true
	})
	b, _ := json.Marshal(_data)
	return string(b)
}
