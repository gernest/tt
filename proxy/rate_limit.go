package proxy

import (
	"context"
	"sync"

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

type rateManager interface {
	Get(ctx context.Context) limit
}

type defaultRateManager struct {
	fill func(RateConfig) limit
	m    map[string]limit
	mu   sync.Mutex
}

func (d *defaultRateManager) Get(ctx context.Context) limit {
	meta := GetContextMeta(ctx)
	r := meta.GetRare()
	if r.Key == "" || r.Average == 0 {
		return noLimit{}
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if r, ok := d.m[r.Key]; ok {
		return r
	}
	n := d.fill(r)
	d.m[r.Key] = n
	return n
}
