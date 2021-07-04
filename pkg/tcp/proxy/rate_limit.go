package proxy

import (
	"context"
	"net"

	"golang.org/x/time/rate"
)

type limit interface {
	WaitN(context.Context, int) error
}

type noLimit struct{}

func (noLimit) WaitN(context.Context, int) error { return nil }

func newRate(v float64) limit {
	if v == 0 {
		return noLimit{}
	}
	return rate.NewLimiter(rate.Limit(v), 0)
}

type RateCopy struct {
	WaitN func(int) error
	net.Conn
	OnWrite func(int)
	OnRead  func(int)
}

func (r *RateCopy) Write(b []byte) (n int, err error) {
	r.OnRead(len(b))
	if err := r.WaitN(len(b)); err != nil {
		return 0, err
	}
	defer func() {
		r.OnWrite(n)
	}()
	n, err = r.Conn.Write(b)
	return
}
