package proxy

import (
	"context"

	"github.com/gernest/tt/api"
)

var _ api.ProxyServer = (*Updates)(nil)

type Updates struct {
	OnInit      func(*api.InitRequest)
	OnUpdate    func(*api.Update)
	OnGetConfig func() (*api.Config, error)
}

func (u *Updates) Init(_ context.Context, x *api.InitRequest) (*api.Response, error) {
	if u.OnInit != nil {
		u.OnInit(x)
	}
	return &api.Response{
		Result: &api.Response_Ok{Ok: true},
	}, nil
}

func (u *Updates) Updates(x api.Proxy_UpdatesServer) error {
	for {
		v, err := x.Recv()
		if err != nil {
			return err
		}
		u.OnUpdate(v)
	}
}

func (u *Updates) GetConfig(context.Context, *api.GetConfigRequest) (*api.Config, error) {
	if u.OnGetConfig != nil {
		return u.OnGetConfig()
	}
	return &api.Config{}, nil
}
