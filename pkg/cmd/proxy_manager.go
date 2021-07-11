package cmd

import (
	"context"

	"github.com/gernest/tt/api"
	"github.com/gernest/tt/pkg/proxy"
)

var _ api.ProxyServer = (*ProxyManager)(nil)

type ProxyManager struct {
	api.UnimplementedProxyServer
	Proxies []proxy.Proxy
}

func New(proxies ...proxy.Proxy) *ProxyManager {
	return &ProxyManager{Proxies: proxies}
}

func (p *ProxyManager) Boot(ctx context.Context, opts *proxy.Options) error {
	for _, v := range p.Proxies {
		if err := v.Boot(ctx, opts); err != nil {
			return err
		}
	}
	return nil
}

func (p *ProxyManager) Close() error {
	for _, v := range p.Proxies {
		if err := v.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (p *ProxyManager) Get(ctx context.Context, req *api.ConfigRequest) (*api.Config, error) {
	c := &api.Config{}
	for _, v := range p.Proxies {
		x, err := v.Config().Get(ctx)
		if err != nil {
			return nil, err
		}
		c.Routes = append(c.Routes, x.Routes...)
	}
	return c, nil
}

func (p *ProxyManager) Put(ctx context.Context, req *api.Config) (*api.Response, error) {
	for _, v := range p.Proxies {
		err := v.Config().Put(ctx, req)
		if err != nil {
			return nil, err
		}
	}
	return &api.Response{
		Result: &api.Response_Ok{Ok: true},
	}, nil
}

func (p *ProxyManager) Post(ctx context.Context, req *api.Config) (*api.Response, error) {
	for _, v := range p.Proxies {
		err := v.Config().Post(ctx, req)
		if err != nil {
			return nil, err
		}
	}
	return &api.Response{
		Result: &api.Response_Ok{Ok: true},
	}, nil
}

func (p *ProxyManager) Delete(ctx context.Context, req *api.DeleteRequest) (*api.Response, error) {
	c := &api.Config{}
	for _, v := range req.Routes {
		c.Routes = append(c.Routes, &api.Route{Name: v})
	}
	for _, v := range p.Proxies {
		err := v.Config().Delete(ctx, c)
		if err != nil {
			return nil, err
		}
	}
	return &api.Response{
		Result: &api.Response_Ok{Ok: true},
	}, nil
}
