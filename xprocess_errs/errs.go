package xprocess_errs

import "errors"

var ErrTimeout = errors.New("xprocess timeout")
var ErrBreak = errors.New("xprocess break")
