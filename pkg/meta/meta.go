package meta

import (
	"context"

	"github.com/gernest/tt/api"
)

type RouteInfo struct {
	Route       *api.Route
	VirtualHost string
}

// Route helper interface for knowing which matched route is being executed.
type Route interface {
	Route() *RouteInfo
}

type metricsKey struct{}

type Metrics struct {
	// Target The upstream target used on the request
	Target string
}

func SetMetric(ctx context.Context, m *Metrics) context.Context {
	return context.WithValue(ctx, metricsKey{}, m)
}

func GetMetics(ctx context.Context) *Metrics {
	return ctx.Value(metricsKey{}).(*Metrics)
}
