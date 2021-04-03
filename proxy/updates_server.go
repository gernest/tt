package proxy

import (
	"context"

	"github.com/gernest/tt/api"
)

var _ api.ProxyServer = (*Updates)(nil)

type Updates struct {
	OnConfigure func(*api.Config) error
}

func (u *Updates) Configure(_ context.Context, x *api.Config) (*api.Response, error) {
	if u.OnConfigure != nil {
		if err := u.OnConfigure(x); err != nil {
			return &api.Response{
				Result: &api.Response_Error{
					Error: err.Error(),
				},
			}, nil
		}
	}
	return &api.Response{
		Result: &api.Response_Ok{Ok: true},
	}, nil
}
