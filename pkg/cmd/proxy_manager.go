package cmd

import (
	"context"

	"github.com/gernest/tt/api"
	"github.com/gernest/tt/pkg/proxy"
	"github.com/hashicorp/raft"
	"go.uber.org/zap"
)

var _ api.ProxyServer = (*ProxyManager)(nil)

type ProxyManager struct {
	api.UnimplementedProxyServer
	Raft    *raft.Raft
	Log     *zap.Logger
	Proxies []proxy.Proxy
}

func (p *ProxyManager) Join(ctx context.Context, in *api.JoinRequest) (*api.JoinResponse, error) {
	log := p.Log.With(
		zap.String("node-id", in.NodeId),
		zap.String("node-address", in.Address),
		zap.String("suffrage", in.Suffrage.String()),
	)
	log.Info("Processing join request")
	configFuture := p.Raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		log.Error("failed to get raft configuration", zap.Error(err))
		return nil, err
	}

	for _, srv := range configFuture.Configuration().Servers {
		// If a node already exists with either the joining node's ID or address,
		// that node may need to be removed from the config first.
		if srv.ID == raft.ServerID(in.NodeId) || srv.Address == raft.ServerAddress(in.Address) {
			// However if *both* the ID and the address are the same, then nothing -- not even
			// a join operation -- is needed.
			if srv.Address == raft.ServerAddress(in.Address) && srv.ID == raft.ServerID(in.NodeId) {
				log.Info("Already a member, ignoring join request")
				return &api.JoinResponse{}, nil
			}

			future := p.Raft.RemoveServer(srv.ID, 0, 0)
			if err := future.Error(); err != nil {
				log.Error("Failed removing existing node", zap.Error(err))
				return nil, err
			}
		}
	}
	var f raft.IndexFuture
	if in.GetSuffrage() == api.JoinRequest_NOT_VOTER {
		f = p.Raft.AddNonvoter(raft.ServerID(in.NodeId), raft.ServerAddress(in.Address), 0, 0)
	} else {
		f = p.Raft.AddVoter(raft.ServerID(in.NodeId), raft.ServerAddress(in.Address), 0, 0)
	}
	if f.Error() != nil {
		return nil, f.Error()
	}
	log.Info("Successfully joined")
	return &api.JoinResponse{}, nil
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
