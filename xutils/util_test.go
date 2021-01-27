package xutils

import (
	"net/http"
	"testing"

	"github.com/pubgo/xerror"
	"github.com/stretchr/testify/assert"
)

type a1 struct {
	a int
}

func a1Test() *a1 {
	return &a1{}
}

func a2Test() *a1 {
	return nil
}

func a3Test() a1 {
	return a1{}
}

func TestAsync(t *testing.T) {
	defer xerror.RespExit()

	var vfn = FuncRaw(http.Get)
	rfn := vfn("https://www.cnblogs.com")
	assert.Equal(t, len(rfn), 2)
}
