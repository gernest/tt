package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"

	"github.com/gernest/tt/api"
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
				Name:   "control,c",
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
				Name:  "load",
				Usage: "If this is provided, routes will be initially loaded from this file",
				Value: "tt.json",
			},
		},
		Action: start,
	}
}

// start starts the proxy service
func start(ctx *cli.Context) error {
	b, err := ioutil.ReadFile(ctx.GlobalString("load"))
	if err != nil {
		zlg.Info(err.Error())
	}
	var c api.Config
	if b != nil {
		var u jsonpb.Unmarshaler
		err := u.Unmarshal(bytes.NewReader(b), &c)
		if err != nil {
			return err
		}
	}

	return startCtx(context.Background(),
		proxy.Options{
			HostPort:        fmt.Sprintf(":%d", ctx.GlobalInt("port")),
			ControlHostPort: fmt.Sprintf(":%d", ctx.GlobalInt("control")),
			AllowedPOrts:    append([]int{ctx.GlobalInt("port")}, ctx.GlobalIntSlice("allowed")...),
			Config:          c,
		},
	)
}

// startCtx starts the proxy and uses port to start the admin RPC
func startCtx(ctx context.Context, o proxy.Options) error {
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