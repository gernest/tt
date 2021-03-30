package proxy

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

type Throttle struct {
	rate    *rate.Limiter
	limit   rate.Limit
	burst   int
	start   int
	current int
	enabled bool
	once    sync.Once
}

func (t *Throttle) init() {}

func (t *Throttle) Take() bool {
	if !t.enabled {
		return true
	}
	if t.current < t.start {
		t.current++
		return true
	}
	t.once.Do(t.init)
	return t.rate.Allow()
}

func (t *Throttle) Wait(ctx context.Context) error {
	return t.rate.Wait(ctx)
}
