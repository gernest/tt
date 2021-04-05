package main

import (
	"os"

	"github.com/gernest/tt/cmd"
	"github.com/gernest/tt/zlg"
	"github.com/urfave/cli"
)

//go:generate protoc -I api/ --go_out=plugins=grpc:./api api/tcp.proto
func main() {
	a := cli.NewApp()
	a.Name = "tt"
	a.Usage = "TCP/UDP -- L4 reverse proxy and load balancer with wasm middlewares "
	a.Commands = append(a.Commands, cmd.Proxy())
	if err := a.Run(os.Args); err != nil {
		zlg.Error(err, "error running the app")
		os.Exit(1)
	}
}
