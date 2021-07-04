package proxy

import (
	"context"
	"os"
	"strings"

	"github.com/gernest/tt/api"
	"github.com/golang/protobuf/jsonpb"
	"github.com/urfave/cli"
)

type Options struct {
	Listen       Listen
	AllowedPorts []int
	Labels       map[string]string
	Config       api.Config
}

func (o Options) Flags() []cli.Flag {
	base := []cli.Flag{
		cli.IntSliceFlag{
			Name:   "allowed_ports",
			EnvVar: "TT_ALLOWED_PORTS",
			Usage:  "Ports that tt is allowed to open",
			Value:  &cli.IntSlice{5700, 5500},
		},
		cli.StringSliceFlag{
			Name:   "labels",
			Usage:  "labels attacked to logs and metrics in the form of key:value",
			EnvVar: "TT_LABELS",
		},
		cli.StringFlag{
			Name:   "config,c",
			Usage:  "path to the routes config file",
			EnvVar: "TT_ROUTES_CONFIG",
		},
	}
	return append(base, o.Listen.Flags()...)
}

func (o *Options) Parse(ctx *cli.Context) error {
	if err := o.Listen.Parse(ctx); err != nil {
		return err
	}
	o.AllowedPorts = ctx.GlobalIntSlice("allowed_ports")
	o.Labels = make(map[string]string)
	for _, v := range ctx.GlobalStringSlice("labels") {
		x := strings.Split(v, ":")
		if len(x) == 2 {
			o.Labels[x[0]] = x[1]
		}
	}
	if c := ctx.GlobalString("config"); c != "" {
		f, err := os.Open(c)
		if err != nil {
			return err
		}
		defer f.Close()
		var u jsonpb.Unmarshaler

		err = u.Unmarshal(f, &o.Config)
		if err != nil {
			return err
		}
	}
	return nil
}

type Listen struct {
	TCP     ListenPort
	HTTP    ListenPort
	Control ListenPort
}

func (l *Listen) Parse(ctx *cli.Context) error {
	//TODO: validate values
	l.TCP.HostPort = ctx.GlobalString("tcp_host_port")
	l.HTTP.HostPort = ctx.GlobalString("http_host_port")
	l.Control.HostPort = ctx.GlobalString("control_host_port")
	return nil
}

func (l Listen) Flags() []cli.Flag {
	return []cli.Flag{
		cli.StringFlag{
			Name:   "tcp_host_port",
			EnvVar: "TT_TCP_HOSTPORT",
			Usage:  "The host:port for serving tcp traffic",
			Value:  ":5700",
		},
		cli.StringFlag{
			Name:   "http_host_port",
			EnvVar: "TT_HTTP_HOSTPORT",
			Usage:  "The host:port for serving http traffic",
			Value:  ":5500",
		},
		cli.StringFlag{
			Name:   "control_host_port",
			EnvVar: "TT_CONTROL_HOSTPORT",
			Usage:  "The host:port for serving gRPC and HTTP control plane",
			Value:  ":5600",
		},
	}
}

type ListenPort struct {
	HostPort string
}

type Proxy interface {
	Config() Config
	// Boot starts the proxy. This must be blocking and only returns when there
	// is an error or when ctx has been cancelled.
	Boot(ctx context.Context, config *Options) error
	Close() error
}

// Config defines methods for dynamic configuration of the proxies.
type Config interface {
	Get(ctx context.Context) (*api.Config, error)
	Put(ctx context.Context, config *api.Config) error
	Post(ctx context.Context, config *api.Config) error
	Delete(ctx context.Context, config *api.Config) error
}
