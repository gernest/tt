package xhttp

import (
	"context"

	"github.com/gernest/tt/api"
	"github.com/gernest/tt/pkg/proxy"
	"google.golang.org/protobuf/proto"
)

var _ proxy.Proxy = (*Proxy)(nil)

type Proxy struct {
	opts   *proxy.Options
	config *api.Config
}

func (p *Proxy) Configure(x *api.Config) error {
	// avoid wasteful reloads by making sure that the configuration changed
	if !proto.Equal(p.config, x) {
		return p.reload(x)
	}
	return nil
}

func (p *Proxy) reload(a *api.Config) error {
	return nil
}

func (p *Proxy) Get(ctx context.Context) (*api.Config, error) {
	return p.config, nil
}

func (p *Proxy) Put(ctx context.Context, config *api.Config) error {
	return p.Configure(config)
}

func clone(a *api.Config) *api.Config {
	return proto.Clone(a).(*api.Config)
}

func (p *Proxy) Post(ctx context.Context, config *api.Config) error {
	old := clone(p.config)
	m := make(map[string]*api.Route)
	for i := 0; i < len(old.Routes); i++ {
		m[old.Routes[i].Name] = old.Routes[i]
	}
	for _, n := range config.Routes {
		if r, ok := m[n.Name]; ok {
			// Update existing route by replacing the old one with the new route.
			*r = *n
		} else {
			old.Routes = append(old.Routes, n)
		}
	}
	return p.Configure(old)
}

func (p *Proxy) Delete(ctx context.Context, config *api.Config) error {
	old := clone(p.config)
	m := make(map[string]*api.Route)
	for i := 0; i < len(old.Routes); i++ {
		m[old.Routes[i].Name] = old.Routes[i]
	}
	for _, n := range config.Routes {
		if r, ok := m[n.Name]; ok {
			delete(m, r.Name)
		}
	}
	x := &api.Config{}
	for _, n := range m {
		x.Routes = append(x.Routes, n)
	}
	return p.Configure(x)
}

func (p *Proxy) Config() proxy.Config {
	return p
}

func (p *Proxy) Boot(ctx context.Context, config *proxy.Options) error {
	return nil
}

func (p *Proxy) Close() error {
	return nil
}
