package proxy

import (
	"context"

	"github.com/gernest/tt/api"
)

type Options struct {
	HostPort        string
	ControlHostPort string
	AllowedPOrts    []int
	Labels          map[string]string
	Config          api.Config
}

type Proxy interface {
	Config() Config
	// Boot starts the proxy. This must be blocking and only returns when there
	// is an error or when ctx has been cancelled.
	Boot(ctx context.Context, config *Options) error
	Close() error
}

// Config defines methods for dynamic configuration of the proxies.
type Config interface {
	Get(ctx context.Context) (*api.Config, error)
	Put(ctx context.Context, config *api.Config) error
	Post(ctx context.Context, config *api.Config) error
	Delete(ctx context.Context, config *api.Config) error
}
