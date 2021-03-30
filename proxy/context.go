package proxy

import (
	"go.uber.org/atomic"
)

var uniq atomic.Int64

// Context holds information about an a tcp request/response
type Context struct {
	ID int64
}

func NewContext() *Context {
	return &Context{ID: uniq.Inc()}
}
