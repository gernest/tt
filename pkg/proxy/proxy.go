package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/dgraph-io/ristretto"
	"github.com/gernest/tt/api"
	"github.com/gernest/tt/pkg/metrics/tseries"
	"github.com/golang/protobuf/jsonpb"
	"github.com/urfave/cli"
)

type Options struct {
	Listen                Listen            `json:",omitempty"`
	AllowedPorts          []int             `json:",omitempty"`
	Labels                map[string]string `json:",omitempty"`
	RoutesPath            string            `json:",omitempty"`
	Cache                 Cache             `json:",omitempty"`
	Info                  Info              `json:",omitempty"`
	DisableHealthEndpoint bool              `json:",omitempty"`
	Metrics               tseries.Config    `json:",omitempty"`
	Wasm                  Wasm              `json:",omitempty"`
	Routes                api.Config        `json:"-"`
}

func (o *Options) setuproutes() error {
	if o.RoutesPath == "" {
		return nil
	}
	var m jsonpb.Unmarshaler
	return filepath.Walk(o.RoutesPath, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".json" {
			return nil
		}
		f, err := os.Open(filepath.Join(o.RoutesPath, path))
		if err != nil {
			return err
		}
		var r api.Route
		err = m.Unmarshal(f, &r)
		if err != nil {
			return err
		}
		o.Routes.Routes = append(o.Routes.Routes, &r)
		return err
	})
}

type Info struct {
	Version   string `json:",omitempty"`
	ReleaseID string `json:",omitempty"`
	ID        string `json:",omitempty"`
}

func (o *Options) Flags() []cli.Flag {
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
			Usage:  "path to configuration file",
			EnvVar: "TT_ROUTES_CONFIG",
		},
		cli.StringFlag{
			Name:   "routes-path",
			Usage:  "path to the routes config file",
			EnvVar: "TT_ROUTES_CONFIG",
		},
	}
	base = append(base, o.Cache.Flags()...)
	base = append(base, o.Metrics.Flags()...)
	base = append(base, o.Wasm.FLags()...)
	return append(base, o.Listen.Flags()...)
}

func (o *Options) Parse(ctx *cli.Context) error {
	if err := o.parse(ctx); err != nil {
		return err
	}
	return o.setuproutes()
}
func (o *Options) parse(ctx *cli.Context) error {
	if c := ctx.GlobalString("config"); c != "" {
		f, err := os.Open(c)
		if err != nil {
			return err
		}
		defer f.Close()
		err = json.NewDecoder(f).Decode(&o)
		if err != nil {
			return err
		}
	}
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
	o.RoutesPath = ctx.GlobalString("routes-path")
	if err := o.Cache.Parse(ctx); err != nil {
		return nil
	}
	if err := o.Metrics.Parse(ctx); err != nil {
		return nil
	}
	if err := o.Wasm.Parse(ctx); err != nil {
		return nil
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

type Cache struct {
	Enabled     bool  `json:",omitempty"`
	NumCounters int64 `json:",omitempty"`
	MaxCost     int64 `json:",omitempty"`
	BufferItems int64 `json:",omitempty"`
	Metrics     bool  `json:",omitempty"`
}

func (c Cache) Config() *ristretto.Config {
	return &ristretto.Config{
		NumCounters: c.NumCounters,
		MaxCost:     c.MaxCost,
		BufferItems: c.BufferItems,
		Metrics:     c.Metrics,
	}
}

func (c *Cache) Parse(ctx *cli.Context) error {
	c.Enabled = ctx.Bool("cache_enabled")
	c.NumCounters = ctx.GlobalInt64("cache_num_counters")
	c.MaxCost = ctx.GlobalInt64("cache_max_cost")
	c.BufferItems = ctx.GlobalInt64("cache_buffer_items")
	c.Metrics = ctx.GlobalBool("cache_metrics")
	return nil
}

func (c Cache) Flags() []cli.Flag {
	return []cli.Flag{
		cli.BoolFlag{
			Name:   "cache_enabled",
			EnvVar: "TT_CACHE_ENABLED",
		},
		cli.Int64Flag{
			Name:   "cache_num_counters",
			EnvVar: "TT_CACHE_NUM_COUNTERS",
			Value:  1e7,
		},
		cli.Int64Flag{
			Name:   "cache_max_cost",
			EnvVar: "TT_CACHE_MAX_COST",
			Value:  1 << 30,
		},
		cli.Int64Flag{
			Name:   "cache_buffer_items",
			EnvVar: "TT_BUFFER_ITEMS",
			Value:  64,
		},
		cli.BoolFlag{
			Name:   "cache_metrics",
			EnvVar: "TT_CACHE_METRICS",
		},
	}
}

type Wasm struct {
	Enabled bool   `json:",omitempty"`
	Dir     string `json:",omitempty"`
}

func (w *Wasm) Parse(ctx *cli.Context) error {
	w.Enabled = ctx.GlobalBoolT("wasm-enabled")
	w.Dir = ctx.GlobalString("wasm-modules-dir")
	if w.Enabled && w.Dir != "" {
		stat, err := os.Stat(w.Dir)
		if err != nil {
			return err
		}
		if !stat.IsDir() {
			return fmt.Errorf("%q is not a directory", w.Dir)
		}
	}
	return nil
}

func (Wasm) FLags() []cli.Flag {
	return []cli.Flag{
		cli.BoolTFlag{
			Name:  "wasm-enabled",
			Usage: "Enables wasm modules",
		},
		cli.StringFlag{
			Name:   "wasm-modules-dir",
			Usage:  "Directory to load wasm modules from",
			EnvVar: "TT_WASM_DIR",
		},
	}
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
