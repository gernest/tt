package meta

import (
	"context"

	"github.com/gernest/tt/api"
)

type RouteInfo struct {
	Route *api.Route
}
type metricsKey struct{}

func SetMetric(ctx context.Context, m *api.AccessEntry) context.Context {
	return context.WithValue(ctx, metricsKey{}, m)
}

func GetMetics(ctx context.Context) *api.AccessEntry {
	return ctx.Value(metricsKey{}).(*api.AccessEntry)
}

type Cleaner interface {
	OnClean(fn func())
	Clean()
}
