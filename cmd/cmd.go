package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/gernest/tt/api"
	proxyPkg "github.com/gernest/tt/pkg/proxy"
	"github.com/gernest/tt/proxy"
	"github.com/gernest/tt/zlg"
	"github.com/golang/protobuf/jsonpb"
	"github.com/urfave/cli"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Proxy returns command for running tcp proxy
func Proxy() cli.Command {
	return cli.Command{
		Name: "proxy",
		Flags: []cli.Flag{
			cli.IntFlag{
				Name:   "port,p",
				Usage:  "Port to bind all tcp traffic",
				EnvVar: "TT_LISTEN_PORT",
				Value:  5555,
			},
			cli.IntFlag{
				Name:   "control",
				Usage:  "gRPC server for realtime dynamic routing updates",
				EnvVar: "TT_CONTROL_PORT",
				Value:  5500,
			},
			cli.IntSliceFlag{
				Name:   "allowed,a",
				Usage:  "List of ports that the tcp proxy is allowed to listen to",
				EnvVar: "TT_ALLOWED_PORTS",
			},
			cli.StringFlag{
				Name:  "config,c",
				Usage: "If this is provided, routes will be initially loaded from this file",
				Value: "tt.json",
			},
		},
		Action: start,
	}
}

// start starts the proxy service
func start(ctx *cli.Context) error {
	file := ctx.String("config")
	b, err := ioutil.ReadFile(file)
	if err != nil {
		zlg.Error(err, "Failed to load config file", zap.String("file", file))
	}
	var c api.Config
	if b != nil {
		var u jsonpb.Unmarshaler
		err := u.Unmarshal(bytes.NewReader(b), &c)
		if err != nil {
			return err
		}
	}

	return StartWithContext(context.Background(),
		&proxyPkg.Options{
			HostPort:        fmt.Sprintf(":%d", ctx.Int("port")),
			ControlHostPort: fmt.Sprintf(":%d", ctx.Int("control")),
			AllowedPOrts:    append([]int{ctx.Int("port")}, ctx.IntSlice("allowed")...),
			Config:          c,
		},
	)
}

// StartWithContext starts the proxy and uses port to start the admin RPC
func StartWithContext(ctx context.Context, o *proxyPkg.Options) error {
	x := proxy.New(ctx, o)
	ls, err := net.Listen("tcp", o.ControlHostPort)
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
