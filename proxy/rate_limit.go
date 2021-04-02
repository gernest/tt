package proxy

import (
	"context"

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
