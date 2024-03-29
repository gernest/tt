package cmd

import (
	"context"
	"fmt"
	"net"

	"github.com/gernest/tt/api"
	"github.com/gernest/tt/pkg/control/cluster"
	proxyPkg "github.com/gernest/tt/pkg/proxy"
	tcpProxy "github.com/gernest/tt/pkg/tcp/proxy"

	"github.com/gernest/tt/pkg/xhttp"
	"github.com/gernest/tt/pkg/zlg"
	"github.com/urfave/cli"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func App(version, commit, date, builtBy string) *cli.App {
	a := cli.NewApp()
	a.Name = "tt"
	a.Version = fmt.Sprintf("%s-%s+%s@%s", version, commit, date, builtBy)
	a.Usage = "TCP/UDP -- L4 reverse proxy and load balancer with wasm middlewares"
	a.Flags = (&proxyPkg.Options{}).Flags()
	a.Action = func(ctx *cli.Context) error {
		return start(ctx, version, commit, date, builtBy)
	}
	return a
}

// start starts the proxy service
func start(ctx *cli.Context, version, commit, date, builtBy string) error {
	opts := &proxyPkg.Options{
		Info: proxyPkg.Info{
			Version:   version,
			ReleaseID: fmt.Sprintf("%s-%s+%s@%s", version, commit, date, builtBy),
		},
	}
	if err := opts.Parse(ctx); err != nil {
		return err
	}
	return StartWithContext(context.Background(), opts)
}

// StartWithContext starts the proxy and uses port to start the admin RPC
func StartWithContext(ctx context.Context, o *proxyPkg.Options) error {

	// add health endpoint
	if !o.DisableHealthEndpoint {
		o.Routes.Routes = append(o.Routes.Routes, xhttp.HealthEndpoint())
	}

	zlg.Info("Setting up fsm for raft", zap.String("node-id", o.Info.ID))
	fsm, err := cluster.NewFSM(o.WorkDir, o.Info.ID)
	if err != nil {
		zlg.Logger.Error("Failed to create fms")
		return err
	}
	defer fsm.Close()
	zlg.Info("Initializing raft cluster", zap.String("node-id", o.Info.ID))
	r, err := cluster.NewRaft(
		o.Bootsrap,
		o.Info.ID,
		o.Listen.Raft.HostPort,
		fsm, o.WorkDir,
	)
	if err != nil {
		zlg.Logger.Error("Failed to create raft cluster")
		return err
	}
	zlg.Info("Successful started raft", zap.String("leader", string(r.Leader())))

	zlg.Info("setting up admin")

	mgr := &ProxyManager{
		Raft: r,
		Log:  zlg.Logger.Named("admin"),
		Proxies: []proxyPkg.Proxy{
			&tcpProxy.Proxy{},
			&xhttp.Proxy{},
		},
	}
	defer mgr.Close()
	ls, err := net.Listen("tcp", o.Listen.Control.HostPort)
	if err != nil {
		return err
	}
	defer ls.Close()
	svr := grpc.NewServer()
	rctx, cancel := context.WithCancel(ctx)
	defer cancel()
	api.RegisterProxyServer(svr, mgr)

	go func() {
		defer cancel()
		zlg.Info("Starting admin rpc sever", zap.String("addr", ls.Addr().String()))
		err := svr.Serve(ls)
		if err != nil {
			zlg.Error(err, "Exit admin rpc server")
		}
	}()
	if err := mgr.Boot(ctx, o); err != nil {
		zlg.Error(err, "Failed to start  proxy server")
	}
	if err := Join(ctx, o); err != nil {
		cancel()
		return err
	}
	<-rctx.Done()
	return nil
}

func Join(ctx context.Context, o *proxyPkg.Options) error {
	if o.Join == "" {
		return nil
	}
	zlg.Info("Joining cluster", zap.String("join_addr", o.Join))
	conn, err := grpc.Dial(o.Join, grpc.WithInsecure())
	if err != nil {
		return err
	}
	x := api.NewProxyClient(conn)
	_, err = x.Join(ctx, &api.JoinRequest{
		NodeId:  o.Info.ID,
		Address: o.Listen.Raft.HostPort,
	})
	return err
}
