package accesslog

import "github.com/urfave/cli"

type Options struct {
	InSize  int
	OutSize int
}

func (o *Options) Parse(ctx *cli.Context) error {
	o.InSize = ctx.GlobalInt("access-log-in-buffer-size")
	o.OutSize = ctx.GlobalInt("access-log-out-buffer-size")
	return nil
}

func (Options) Flags() []cli.Flag {
	return []cli.Flag{
		cli.IntFlag{
			Name:   "access-log-in-buffer-size",
			EnvVar: "TT_ACCESS_LOG_IN_BUFFER_SIZE",
			Value:  100,
		},
		cli.IntFlag{
			Name:   "access-log-out-buffer-size",
			EnvVar: "TT_ACCESS_LOG_IN_BUFFER_SIZE",
			Value:  100,
		},
	}
}
