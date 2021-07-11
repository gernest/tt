package proxy

import (
	"context"
	"net"
	"sync"

	"github.com/gernest/tt/api"
	proxyPkg "github.com/gernest/tt/pkg/proxy"
	"github.com/gernest/tt/pkg/tcp"
	"github.com/gernest/tt/pkg/tcp/middlewares"
	"github.com/gernest/tt/pkg/zlg"
	"github.com/golang/protobuf/ptypes"
	"github.com/smallnest/weighted"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type configMap map[string]*tcp.Config

func (m configMap) get(ipPort string) *tcp.Config {
	if c, ok := m[ipPort]; ok {
		return c
	}
	c := &tcp.Config{}
	m[ipPort] = c
	return c
}

func (m configMap) addRoute(ipPort string, r tcp.Route) {
	cfg := m.get(ipPort)
	cfg.Routes = append(cfg.Routes, r)
}

// AddSNIMatchRoute appends a route to the ipPort listener that routes
// to dest if the incoming TLS SNI server name is accepted by
// matcher. If it doesn't match, rule processing continues for any
// additional routes on ipPort.
//
// By default, the proxy will route all ACME tls-sni-01 challenges
// received on ipPort to all SNI dests. You can disable ACME routing
// with AddStopACMESearch.
//
// The ipPort is any valid net.Listen TCP address.
func (m configMap) AddSNIMatchRoute(ipPort string, matcher middlewares.Matcher, dest tcp.Target) {
	cfg := m.get(ipPort)
	if cfg.AllowACME {
		if len(cfg.AcmeTargets) == 0 {
			cfg.Routes = append(cfg.Routes, &middlewares.AcmeMatch{cfg})
		}
		cfg.AcmeTargets = append(cfg.AcmeTargets, dest)
	}
	cfg.Routes = append(cfg.Routes, middlewares.SniMatch{matcher, dest})
}

// AddSNIRoute appends a route to the ipPort listener that routes to
// dest if the incoming TLS SNI server name is sni. If it doesn't
// match, rule processing continues for any additional routes on
// ipPort.
//
// By default, the proxy will route all ACME tls-sni-01 challenges
// received on ipPort to all SNI dests. You can disable ACME routing
// with AddStopACMESearch.
//
// The ipPort is any valid net.Listen TCP address.
func (m configMap) AddSNIRoute(ipPort, sni string, dest tcp.Target) {
	m.AddSNIMatchRoute(ipPort, middlewares.Equals(sni), dest)
}

// AddRoute appends an always-matching route to the ipPort listener,
// directing any connection to dest.
//
// This is generally used as either the only rule (for simple TCP
// proxies), or as the final fallback rule for an ipPort.
//
// The ipPort is any valid net.Listen TCP address.
func (m configMap) AddRoute(ipPort string, dest tcp.Target) {
	m.addRoute(ipPort, fixedTarget{dest})
}

func (m configMap) AddStopACMESearch(ipPort string) {
	m.get(ipPort).AllowACME = false
}

func (m configMap) AddAllowACMESearch(ipPort string) {
	m.get(ipPort).AllowACME = true
}

// defaultIPPort used to map to the default ip:port
const defaultIPPort = "dream"
const defaultNetwork = "tcp"

// Route generates configuration based on r
func (m configMap) Route(r *api.Route) {
	var labels []zapcore.Field
	for k, v := range r.MetricsLabels {
		labels = append(labels, zap.String(k, v))
	}
	ipPort := proxyPkg.BindToHostPort(r.Bind, defaultIPPort)
	network := defaultNetwork
	labels = append(labels, zap.String("ip:port", ipPort))
	zlg.Info("Loading route", labels...)
	m.get(ipPort).AllowACME = r.AllowAcme
	m.get(ipPort).Network = network
	switch e := r.Condition.Match.(type) {
	case *api.RequestMatch_Sni:
		zlg.Info("Adding sni route",
			zap.String("host", e.Sni),
		)
		m.AddSNIRoute(ipPort, e.Sni, buildTarget(r))
	case *api.RequestMatch_Fixed:
		zlg.Info("Adding fixed route")
		m.AddRoute(ipPort, buildTarget(r))
	}
}

func buildTarget(r *api.Route) tcp.Target {
	return tcp.BuildMiddlewares(r).Then(target(r))
}

func target(r *api.Route) tcp.Target {
	if r.LoadBalance != nil {
		switch r.LoadBalanceAlgo {
		case api.Route_RoundRobinWeighted:
			w := &weighted.RRW{}
			for _, v := range r.LoadBalance {
				t := toDial(v, r)
				w.Add(t, int(v.Weight))
			}
			return &balance{
				ba: balanceFn(func() tcp.Target {
					return w.Next().(tcp.Target)
				}),
			}
		case api.Route_SmoothWeighted:
			w := &weighted.SW{}
			for _, v := range r.LoadBalance {
				t := toDial(v, r)
				w.Add(t, int(v.Weight))
			}
			return &balance{
				ba: balanceFn(func() tcp.Target {
					return w.Next().(tcp.Target)
				}),
			}

		case api.Route_RandomWeighted:
			w := &weighted.RandW{}
			for _, v := range r.LoadBalance {
				t := toDial(v, r)
				w.Add(t, int(v.Weight))
			}
			return &balance{
				ba: balanceFn(func() tcp.Target {
					return w.Next().(tcp.Target)
				}),
			}
		}
	}
	return nil
}

func toDial(a *api.WeightedAddr, r *api.Route) *DialProxy {
	var ipPort string
	network := defaultNetwork
	if a.Addr != nil {
		ipPort = a.Addr.Address
		network = a.Addr.Network
	}
	timeout, _ := ptypes.Duration(r.Timeout)
	keepAlive, _ := ptypes.Duration(r.KeepAlive)
	if r.EnableOptimizedCopy {
	}
	var up, down Speed
	if speed := r.Speed; speed != nil {
		up = Speed(speed.Upstream)
		down = Speed(speed.Downstream)
	}
	return &DialProxy{
		Network:         network,
		Addr:            ipPort,
		DialTimeout:     timeout,
		KeepAlivePeriod: keepAlive,
		MetricsLabels:   a.MetricLabels,
		UpstreamSpeed:   up,
		DownstreamSpeed: down,
	}
}

type balancer interface {
	Next() tcp.Target
}

type balanceFn func() tcp.Target

func (fn balanceFn) Next() tcp.Target {
	return fn()
}

var _ tcp.Target = (*balance)(nil)

type balance struct {
	ba balancer
	mu sync.Mutex
}

func (b *balance) HandleConn(ctx context.Context, conn net.Conn) {
	b.mu.Lock()
	t := b.ba.Next()
	b.mu.Unlock()
	t.HandleConn(ctx, conn)
}
