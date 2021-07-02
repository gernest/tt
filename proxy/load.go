package proxy

import (
	"context"
	"net"
	"sync"

	"github.com/gernest/tt/api"
	"github.com/gernest/tt/pkg/tcp"
	"github.com/gernest/tt/zlg"
	"github.com/golang/protobuf/ptypes"
	"github.com/smallnest/weighted"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type configMap map[string]*config

func (m configMap) get(ipPort string) *config {
	if c, ok := m[ipPort]; ok {
		return c
	}
	c := &config{}
	m[ipPort] = c
	return c
}

func (m configMap) addRoute(ipPort string, r tcp.Route) {
	cfg := m.get(ipPort)
	cfg.routes = append(cfg.routes, r)
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
func (m configMap) AddSNIMatchRoute(ipPort string, matcher Matcher, dest tcp.Target) {
	cfg := m.get(ipPort)
	if cfg.allowACME {
		if len(cfg.acmeTargets) == 0 {
			cfg.routes = append(cfg.routes, &acmeMatch{cfg})
		}
		cfg.acmeTargets = append(cfg.acmeTargets, dest)
	}
	cfg.routes = append(cfg.routes, sniMatch{matcher, dest})
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
	m.AddSNIMatchRoute(ipPort, equals(sni), dest)
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
	m.get(ipPort).allowACME = false
}

func (m configMap) AddAllowACMESearch(ipPort string) {
	m.get(ipPort).allowACME = true
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
	ipPort := defaultIPPort
	network := defaultNetwork
	if r.Src != nil {
		if r.Src.Address != "" {
			ipPort = r.Src.Address
		}
		if r.Src.Network != "" {
			ipPort = r.Src.Network
		}
	}
	labels = append(labels, zap.String("ip:port", ipPort))
	zlg.Info("Loading route", labels...)
	m.get(ipPort).allowACME = r.AllowAcme
	m.get(ipPort).network = network
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
	return buildMiddleares(r).then(target(r))
}

func target(r *api.Route) tcp.Target {
	if r.Endpoint != nil {
		return toDial(r.Endpoint, r)
	}
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
