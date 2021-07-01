package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/gernest/tt/cmd"
	"github.com/gernest/tt/zlg"
	"github.com/urfave/cli"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
	builtBy = ""
)

//go:generate protoc -I api/ --go_out=./api --go_opt=paths=source_relative  --go-grpc_out=./api --go-grpc_opt=paths=source_relative api/tcp.proto
func main() {
	a := cli.NewApp()
	a.Name = "tt"
	a.Version = buildVersion(version, commit, date, builtBy)
	a.Usage = "TCP/UDP -- L4 reverse proxy and load balancer with wasm middlewares "
	a.Commands = append(a.Commands, cmd.Proxy())
	if err := a.Run(os.Args); err != nil {
		zlg.Error(err, "error running the app")
		os.Exit(1)
	}
}

func buildVersion(version, commit, date, builtBy string) string {
	result := version
	if commit != "" {
		result = fmt.Sprintf("%s\ncommit: %s", result, commit)
	}
	if date != "" {
		result = fmt.Sprintf("%s\nbuilt at: %s", result, date)
	}
	if builtBy != "" {
		result = fmt.Sprintf("%s\nbuilt by: %s", result, builtBy)
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Sum != "" {
		result = fmt.Sprintf("%s\nmodule version: %s, checksum: %s", result, info.Main.Version, info.Main.Sum)
	}
	return result
}
