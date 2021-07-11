package main

import (
	"os"

	"github.com/gernest/tt/pkg/cmd"
	"github.com/gernest/tt/pkg/zlg"
)

var (
	version = "dev"
	commit  = ""
	date    = ""
	builtBy = ""
)

//go:generate protoc -I api/ --go_out=./api --go_opt=paths=source_relative  --go-grpc_out=./api --go-grpc_opt=paths=source_relative api/tcp.proto
func main() {
	a := cmd.App(version, commit, date, builtBy)
	if err := a.Run(os.Args); err != nil {
		zlg.Error(err, "error running the app")
		os.Exit(1)
	}
}
