package proxy

import (
	"context"

	"golang.org/x/time/rate"
)

type Throttle struct {
	rate    *rate.Limiter
	enabled bool
}

func (t *Throttle) Wait(ctx context.Context, size int) error {
	if !t.enabled {
		return nil
	}
	return t.rate.WaitN(ctx, size)
}
