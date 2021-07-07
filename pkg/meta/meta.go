package meta

import (
	"context"

	"github.com/gernest/tt/api"
)

type RouteInfo struct {
	Route       *api.Route
	VirtualHost string
}

type metricsKey struct{}

type Metrics struct {
	Target      string
	Service     string
	Route       string
	VirtualHost string
}

func SetMetric(ctx context.Context, m *Metrics) context.Context {
	return context.WithValue(ctx, metricsKey{}, m)
}

func GetMetics(ctx context.Context) *Metrics {
	return ctx.Value(metricsKey{}).(*Metrics)
}
