package xprocess

import "github.com/pubgo/xprocess/xprocess_group"

func NewGroup(c ...uint16) *xprocess_group.Group { return xprocess_group.New(c...) }
