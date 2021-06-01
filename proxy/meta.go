package proxy

import (
	"context"
	"time"

	"go.uber.org/atomic"
)

type Protocol uint

const (
	RawTCP Protocol = iota
	HTTP
	UDP
	Websocket
)

func (p Protocol) String() string {
	switch p {
	case HTTP:
		return "http"
	case UDP:
		return "udp"
	case Websocket:
		return "websocket"
	default:
		return "tcp"
	}
}

var contextID atomic.Int64

// Meta a lot of details that is passed around with the  connection.
type ContextMeta struct {
	ID atomic.Int64
	// Downstream
	D TCP
	// Upstream
	U TCP
	// ACME true if we are serving acme challenge
	ACME atomic.Bool
	// Fixed is true if we are serving a fixed target
	Fixed atomic.Bool
	// NoMatch true if we have no matching route
	NoMatch atomic.Bool
	// ServerName SNI or Host of the server serving the request
	ServerName atomic.String
	// RouteName the name of the route if any
	RouteName atomic.String
	// Protocol The protocol which we are serving
	Protocol   atomic.Uint32
	Start, End time.Time
	Speed      SpeedRateConfig
	Rate       Rate
	// Labels these are labels that are attached to the request
	Labels map[string]string
}

// GetRare returns rate limiting configuration for this route
func (m ContextMeta) GetRare() RateConfig {
	var key string
	switch RateBy(m.Rate.By.Load()) {
	case Host:
		key = m.ServerName.Load()
	}
	return RateConfig{
		Route:   m.RouteName.Load(),
		Key:     key,
		Average: m.Rate.Average.Load(),
		Burst:   int(m.Rate.Burst.Load()),
	}
}

type RateConfig struct {
	Route   string
	Key     string
	Average float64
	Burst   int
}

type SpeedRateConfig struct {
	// This is the amount of bytes read from the client by the server per second.
	// You can think of this as limit on upload speed, where tt is the server
	// receiving the data so it will use this value to limit how much it will be
	// reading per second
	// You can use this to control upload speeds
	Downstream atomic.Float64

	// This is the amount of bytes read from the proxied(upstream) end point. In this case
	// tt will use this to limit how much data it reads from upstream. You can use
	// this to control download speeds
	Upstream atomic.Float64
}

type RateBy uint

const (
	IP RateBy = iota
	Host
)

type Rate struct {
	By      atomic.Uint32
	Average atomic.Float64
	Burst   atomic.Int64
}

// TCP stats about a tcp socket connection
type TCP struct {
	// A socket address
	A Address
	// R bytes read from this socket.
	R atomic.Int64
	// W bytes written to this socket
	W atomic.Int64
}

type metakey struct{}

// Address data for connection address
type Address struct {
	//L Local
	L Addr
	// R remote
	R Addr
}

type Addr struct {
	Address string
}

// UpdateContext applies fn to the Meta that is in ctx and returns a new context if ctx
// doesn't have Meta yet.
func UpdateContext(ctx context.Context, fn func(*ContextMeta)) context.Context {
	if x := ctx.Value(metakey{}); x != nil {
		v := x.(*ContextMeta)
		fn(v)
		return ctx
	}
	var m ContextMeta
	m.ID.Store(contextID.Inc())
	m.Start = time.Now()
	fn(&m)
	return context.WithValue(ctx, metakey{}, &m)
}

func GetContextMeta(ctx context.Context) *ContextMeta {
	if m := ctx.Value(metakey{}); m != nil {
		return m.(*ContextMeta)
	}
	return &ContextMeta{}
}
