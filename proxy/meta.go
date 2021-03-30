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

// Meta a lot of details that is passed around with the  connection.
type ContextMeta struct {
	// Request
	R TCP
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
	// Protocol The protocol which we are serving
	Protocol atomic.Uint32
	Start    time.Time
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
	Network string
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
	m.Start = time.Now()
	fn(&m)
	return context.WithValue(ctx, metakey{}, &m)
}

func Update(ctx context.Context, fn func(*ContextMeta)) {
	if x := ctx.Value(metakey{}); x != nil {
		v := x.(*ContextMeta)
		fn(v)
	}
}
