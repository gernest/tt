package proxy

import (
	"context"

	"github.com/gernest/tt/api"
)

type Configuration struct{}

type Proxy interface {
	Config() Config
	// Start starts the proxy. This must be blocking and only returns when there
	// is an error or when ctx has been cancelled.
	Start(ctx context.Context, config Configuration) error
	Close(ctx context.Context) error
}

// Config defines methods for dynamic configuration of the proxies.
type Config interface {
	Get(ctx context.Context) (*api.Config, error)
	AddRoute(ctx context.Context, route *api.Route) error
	UpdateRoute(ctx context.Context, route *api.Route) error
	DeleteRoute(ctx context.Context, route *api.Route) error
}
