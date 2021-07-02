package tcp

import (
	"bufio"
	"context"
	"net"
)

// Target is what an incoming matched connection is sent to.
type Target interface {
	// HandleConn is called when an incoming connection is
	// matched. After the call to HandleConn, the tcpproxy
	// package never touches the conn again. Implementations are
	// responsible for closing the connection when needed.
	//
	// The concrete type of conn will be of type *Conn if any
	// bytes have been consumed for the purposes of route
	// matching.
	HandleConn(context.Context, net.Conn)
}

// A Route matches a connection to a target.
type Route interface {
	// Match examines the initial bytes of a connection, looking for a
	// Match. If a Match is found, Match returns a non-nil Target to
	// which the stream should be proxied. Match returns nil if the
	// connection doesn't Match.
	//
	// Match must not consume bytes from the given bufio.Reader, it
	// can only Peek.
	//
	// If an sni or host header was parsed successfully, that will be
	// returned as the second parameter.
	Match(context.Context, *bufio.Reader) (Target, string)
}
