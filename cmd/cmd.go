package cmd

import (
	"context"
	"fmt"
	"net"

	"github.com/gernest/tt/api"
	proxyPkg "github.com/gernest/tt/pkg/proxy"
	"github.com/gernest/tt/pkg/tcp/proxy"
	"github.com/gernest/tt/zlg"
	"github.com/urfave/cli"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func App(version, commit, date, builtBy string) *cli.App {
	a := cli.NewApp()
	a.Name = "tt"
	a.Version = fmt.Sprintf("%s-%s+%s@%s", version, commit, date, builtBy)
	a.Usage = "TCP/UDP -- L4 reverse proxy and load balancer with wasm middlewares"
	a.Flags = proxyPkg.Options{}.Flags()
	a.Action = start
	return a
}

// start starts the proxy service
func start(ctx *cli.Context) error {
	opts := &proxyPkg.Options{}
	if err := opts.Parse(ctx); err != nil {
		return err
	}
	return StartWithContext(context.Background(), opts)
}

// StartWithContext starts the proxy and uses port to start the admin RPC
func StartWithContext(ctx context.Context, o *proxyPkg.Options) error {
	x := proxy.New(ctx, o)
	ls, err := net.Listen("tcp", o.Listen.Control.HostPort)
	if err != nil {
		return err
	}
	defer ls.Close()
	svr := grpc.NewServer()
	rctx, cancel := context.WithCancel(ctx)
	api.RegisterProxyServer(svr, x.RPC())
	go func() {
		defer cancel()
		zlg.Info("Starting admin rpc sever", zap.String("addr", ls.Addr().String()))
		err := svr.Serve(ls)
		if err != nil {
			zlg.Error(err, "Exit admin rpc server")
		}
	}()
	go func() {
		if err := x.Start(); err != nil {
			zlg.Error(err, "Failed to start  proxy server")
		}
	}()
	<-rctx.Done()
	return nil
}
