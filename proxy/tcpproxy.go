// Copyright 2017 Google Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package proxy

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/gernest/tt/api"
	"github.com/gernest/tt/zlg"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

// ErrPortNotAllowed is returned when opening non whitelisted port.
var ErrPortNotAllowed = errors.New("proxy: port not allowed")

// Proxy is a proxy. Its zero value is a valid proxy that does
// nothing. Call methods to add routes before calling Start or Run.
//
// The order that routes are added in matters; each is matched in the order
// registered.
type Proxy struct {
	configMap

	// availablePorts is updatable via the admin api
	availablePorts []int32
	config         api.Config

	lns    map[string]net.Listener
	cancel context.CancelFunc
	mu     sync.RWMutex
	// ListenFunc optionally specifies an alternate listen
	// function. If nil, net.Dial is used.
	// The provided net is always "tcp".
	ListenFunc func(net, laddr string) (net.Listener, error)

	ctx context.Context

	// The host:ip on which this host is listening from.
	opts Options
}

// goodPort returns true if port is good and should be ok to listen on.
func (p *Proxy) goodPort(port int) bool {
	for _, v := range p.opts.AllowedPOrts {
		if v == port {
			return true
		}
	}
	for _, v := range p.availablePorts {
		if v == int32(port) {
			return true
		}
	}
	return false
}

type Options struct {
	HostPort        string
	ControlHostPort string
	AllowedPOrts    []int
	Labels          map[string]string
	Config          api.Config
}

func New(ctx context.Context, opts Options) *Proxy {
	conf := make(configMap)
	x := conf.get(opts.HostPort)
	x.routes = append(x.routes, noopRoute{})
	for _, r := range opts.Config.Routes {
		conf.Route(r)
	}
	return &Proxy{
		configMap: conf,
		lns:       make(map[string]net.Listener),
		ctx:       ctx,
		opts:      opts,
	}
}

// RPC returns rpc server used to dynamically update the state of the proxy
func (p *Proxy) RPC() *Updates {
	return &Updates{
		OnConfigure: p.Configure,
	}
}

func (p *Proxy) Configure(x *api.Config) error {
	// avoid wasteful reloads by making sure that the configuration changed
	if !proto.Equal(&p.config, x) {
		m := make(configMap)
		for _, r := range x.Routes {
			m.Route(r)
		}
		if err := p.Reload(m); err != nil {
			// restore old apis because we can't load the new ones
			if n := p.TriggerReload(); n != nil {
				// TODO we are in a broken state log/error or something
			}
			return err
		}
		p.config = *x
	}
	return nil
}

func (p *Proxy) TriggerReload() error {
	m := make(configMap)
	for _, r := range p.config.Routes {
		m.Route(r)
	}
	return p.Reload(m)
}

func (p *Proxy) GetConfig() (*api.Config, error) {
	return &p.config, nil
}

// Matcher reports whether hostname matches the Matcher's criteria.
type Matcher func(ctx context.Context, hostname string) bool

// equals is a trivial Matcher that implements string equality.
func equals(want string) Matcher {
	return func(_ context.Context, got string) bool {
		return want == got
	}
}

// config contains the proxying state for one listener.
type config struct {
	routes      []route
	acmeTargets []Target // accumulates targets that should be probed for acme.
	allowACME   bool     // if true, AddSNIRoute doesn't add targets to acmeTargets.
	network     string
}

// A route matches a connection to a target.
type route interface {
	// match examines the initial bytes of a connection, looking for a
	// match. If a match is found, match returns a non-nil Target to
	// which the stream should be proxied. match returns nil if the
	// connection doesn't match.
	//
	// match must not consume bytes from the given bufio.Reader, it
	// can only Peek.
	//
	// If an sni or host header was parsed successfully, that will be
	// returned as the second parameter.
	match(context.Context, *bufio.Reader) (Target, string)
}

func (p *Proxy) netListen() func(net, laddr string) (net.Listener, error) {
	if p.ListenFunc != nil {
		return p.ListenFunc
	}
	return net.Listen
}

type fixedTarget struct {
	t Target
}

func (m fixedTarget) match(ctx context.Context, r *bufio.Reader) (Target, string) {
	meta := GetContextMeta(ctx)
	meta.Fixed.Store(true)
	return m.t, ""
}

// Close closes all the proxy's self-opened listeners.
func (p *Proxy) Close() error {
	if p.cancel != nil {
		p.cancel()
	}
	for _, c := range p.lns {
		c.Close()
	}
	return nil
}

// port Returns host:port if configPort is default we retrun the p.hostPort
func (p *Proxy) port(configPort string) string {
	if configPort == defaultIPPort {
		return p.opts.HostPort
	}
	return configPort
}

// Start creates a TCP listener for each unique ipPort from the
// previously created routes and starts the proxy. It returns any
// error from starting listeners.
//
// If it returns a non-nil error, any successfully opened listeners
// are closed.
func (p *Proxy) Start() (err error) {
	defer func() {
		if err != nil {
			p.Close()
		}
	}()
	if p.lns == nil {
		p.lns = make(map[string]net.Listener)
	}
	if p.cancel != nil {
		p.cancel()
	}
	if len(p.configMap) == 0 {
		zlg.Info("No routes configured yet")
		return nil
	}

	// Update mapping by replacing defaultIPPort with actual hos:port that was
	// configured in the Proxy instance
	//
	// From here onward we are dealing with host:port only so no need for special
	// handling of defaultIPPort
	for k, v := range p.configMap {
		if k == defaultIPPort {
			delete(p.configMap, k)
			p.configMap[p.port(k)] = v
		}
	}

	ctx, cancel := context.WithCancel(p.ctx)
	p.cancel = cancel
	zlg.Info("Starting Proxy", zap.String("allowed-ports", fmt.Sprint(p.opts.AllowedPOrts)))

	set := make(map[string]struct{})
	for ipPort := range p.configMap {
		set[ipPort] = struct{}{}
	}

	// close all liseneres that are not part of the new set. If a new set of
	// host:port arrives we need to cleanup all dangling listeners of deleted
	// host:ip routes
	for ls := range p.lns {
		if _, ok := set[ls]; !ok {
			zlg.Info("Deleting listener", zap.String("ip_port", ls))
			p.lns[ls].Close()
			delete(p.lns, ls)
		}
	}

	for hostPort := range set {
		if _, ok := p.lns[hostPort]; !ok {
			var port string
			_, port, err = net.SplitHostPort(hostPort)
			if err != nil {
				zlg.Error(err, "Failed to split ip:port", zap.String("ip:port", hostPort))
				return
			}
			px, _ := strconv.Atoi(port)
			if !p.goodPort(px) {
				zlg.Error(ErrPortNotAllowed, "Trying to open blacklisted port",
					zap.Int("port", px),
					zap.String("ip:port", hostPort),
				)
				continue
			}
			var ln net.Listener
			ln, err = p.netListen()("tcp", hostPort)
			if err != nil {
				return
			}
			zlg.Info("Started listener", zap.String("addr", hostPort))
			p.lns[hostPort] = ln
		}
	}
	// start serving traffic
	p.start(ctx)
	return nil
}

// ServerInfo general information about the server.
type ServerInfo struct {
	Listener net.Listener
	Proxy    *Proxy
}

type serverInfoKey struct{}

// GetInfo returns server information from context
func GetInfo(ctx context.Context) *ServerInfo {
	if v := ctx.Value(serverInfoKey{}); v != nil {
		return v.(*ServerInfo)
	}
	return nil
}

func (p *Proxy) base(ctx context.Context, ls net.Listener) context.Context {
	return context.WithValue(ctx, serverInfoKey{}, &ServerInfo{
		Listener: ls,
		Proxy:    p,
	})
}

func (p *Proxy) start(ctx context.Context) {
	for x, ln := range p.lns {
		go p.serveListener(ctx, ln, x)
	}
}

func (p *Proxy) serveListener(ctx context.Context, ln net.Listener, hostPort string) {
	x := p.configMap[hostPort]
	zlg.Info("Start Listening for traffic", zap.String("host:port", hostPort))
	for {
		if ctx.Err() != nil {
			return
		}
		c, err := ln.Accept()
		if err != nil {
			if !ErrIsNetClosed(err) {
				zlg.Error(err, "Failed to accept connection")
			}
			return
		}
		zlg.Info(fmt.Sprintf("%s --> %s", c.RemoteAddr().String(), c.LocalAddr().String()))
		base := p.base(ctx, ln)
		base = UpdateContext(base, func(m *ContextMeta) {
			m.D.A.L.Address = c.LocalAddr().String()
			m.D.A.R.Address = c.RemoteAddr().Network()
		})
		go serveConn(base, c, x.routes)
	}
}

// ErrIsNetClosed returns true if err is an error returned when using a closed
// network connection
func ErrIsNetClosed(err error) bool {
	var e *net.OpError
	if errors.As(err, &e) {
		return e.Err.Error() == "use of closed network connection"
	}
	return false
}

type noopRoute struct{}

func (noopRoute) match(context.Context, *bufio.Reader) (Target, string) {
	return nil, ""
}

// serveConn runs in its own goroutine and matches c against routes.
// It returns whether it matched purely for testing.
func serveConn(ctx context.Context, c net.Conn, routes []route) bool {
	br := bufio.NewReader(c)
	for _, route := range routes {
		if target, hostName := route.match(ctx, br); target != nil {
			if n := br.Buffered(); n > 0 {
				peeked, _ := br.Peek(br.Buffered())
				c = &Conn{
					HostName: hostName,
					Peeked:   peeked,
					Conn:     c,
				}
			}
			target.HandleConn(ctx, c)
			return true
		}
	}
	meta := GetContextMeta(ctx)
	meta.NoMatch.Store(true)
	zlg.Info("no routes matched conn",
		zap.String("remote_addr", c.RemoteAddr().String()),
		zap.String("local_addr", c.LocalAddr().String()),
	)
	c.Close()
	return false
}

func (p *Proxy) Reload(m configMap) error {
	zlg.Info("Reloading")
	p.mu.Lock()
	p.configMap = m
	p.mu.Unlock()
	return p.Start()
}

// Conn is an incoming connection that has had some bytes read from it
// to determine how to route the connection. The Read method stitches
// the peeked bytes and unread bytes back together.
type Conn struct {
	// HostName is the hostname field that was sent to the request router.
	// In the case of TLS, this is the SNI header, in the case of HTTPHost
	// route, it will be the host header.  In the case of a fixed
	// route, i.e. those created with AddRoute(), this will always be
	// empty. This can be useful in the case where further routing decisions
	// need to be made in the Target implementation.
	HostName string

	// Peeked are the bytes that have been read from Conn for the
	// purposes of route matching, but have not yet been consumed
	// by Read calls. It set to nil by Read when fully consumed.
	Peeked []byte

	// Conn is the underlying connection.
	// It can be type asserted against *net.TCPConn or other types
	// as needed. It should not be read from directly unless
	// Peeked is nil.
	net.Conn
}

func (c *Conn) Read(p []byte) (n int, err error) {
	if len(c.Peeked) > 0 {
		n = copy(p, c.Peeked)
		c.Peeked = c.Peeked[n:]
		if len(c.Peeked) == 0 {
			c.Peeked = nil
		}
		return n, nil
	}
	return c.Conn.Read(p)
}

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

type Incoming struct {
	*bufio.Reader
}

// To is shorthand way of writing &tlsproxy.DialProxy{Addr: addr}.
func To(addr string) *DialProxy {
	return &DialProxy{Addr: addr}
}

// DialProxy implements Target by dialing a new connection to Addr
// and then proxying data back and forth.
//
// The To func is a shorthand way of creating a DialProxy.
type DialProxy struct {
	Network string

	// Addr is the TCP address to proxy to.
	Addr string

	// KeepAlivePeriod sets the period between TCP keep alives.
	// If zero, a default is used. To disable, use a negative number.
	// The keep-alive is used for both the client connection and
	KeepAlivePeriod time.Duration

	// DialTimeout optionally specifies a dial timeout.
	// If zero, a default is used.
	// If negative, the timeout is disabled.
	DialTimeout time.Duration

	// DialContext optionally specifies an alternate dial function
	// for TCP targets. If nil, the standard
	// net.Dialer.DialContext method is used.
	DialContext func(ctx context.Context, network, address string) (net.Conn, error)

	// OnDialError optionally specifies an alternate way to handle errors dialing Addr.
	// If nil, the error is logged and src is closed.
	// If non-nil, src is not closed automatically.
	OnDialError func(src net.Conn, dstDialErr error)

	// ProxyProtocolVersion optionally specifies the version of
	// HAProxy's PROXY protocol to use. The PROXY protocol provides
	// connection metadata to the DialProxy target, via a header
	// inserted ahead of the client's traffic. The DialProxy target
	// must explicitly support and expect the PROXY header; there is
	// no graceful downgrade.
	// If zero, no PROXY header is sent. Currently, version 1 is supported.
	ProxyProtocolVersion int
	// MetricsLabels labels included when emitting metrics about the TPC proxying
	// with this Dial
	MetricsLabels map[string]string

	UpstreamSpeed   Speed
	DownstreamSpeed Speed
}

// UnderlyingConn returns c.Conn if c of type *Conn,
// otherwise it returns c.
func UnderlyingConn(c net.Conn) net.Conn {
	if wrap, ok := c.(*Conn); ok {
		return wrap.Conn
	}
	return c
}

func goCloseConn(c net.Conn) { go c.Close() }

// HandleConn implements the Target interface.
func (dp *DialProxy) HandleConn(ctx context.Context, src net.Conn) {
	meta := GetContextMeta(ctx)
	// we update sppeds that were set on this dial
	up, _ := dp.UpstreamSpeed.Limit()
	//TOD log error
	meta.Speed.Upstream.Store(up)
	down, _ := dp.DownstreamSpeed.Limit()
	//TOD log error
	meta.Speed.Downstream.Store(down)
	var cancel context.CancelFunc
	if dp.DialTimeout >= 0 {
		ctx, cancel = context.WithTimeout(ctx, dp.dialTimeout())
	}
	network := defaultNetwork
	if dp.Network != "" {
		network = dp.Network
	}
	dst, err := dp.dialContext()(ctx, network, dp.Addr)
	if cancel != nil {
		cancel()
	}
	if err != nil {
		dp.onDialError()(src, err)
		return
	}
	defer goCloseConn(dst)

	if err = dp.sendProxyHeader(dst, src); err != nil {
		dp.onDialError()(src, err)
		return
	}
	defer goCloseConn(src)

	if ka := dp.keepAlivePeriod(); ka > 0 {
		if c, ok := UnderlyingConn(src).(*net.TCPConn); ok {
			c.SetKeepAlive(true)
			c.SetKeepAlivePeriod(ka)
		}
		if c, ok := dst.(*net.TCPConn); ok {
			c.SetKeepAlive(true)
			c.SetKeepAlivePeriod(ka)
		}
	}
	errc := make(chan error, 1)
	go proxyCopy(errc, src, dst)
	go proxyCopy(errc, dst, src)
	<-errc
}

func (dp *DialProxy) sendProxyHeader(w io.Writer, src net.Conn) error {
	switch dp.ProxyProtocolVersion {
	case 0:
		return nil
	case 1:
		var srcAddr, dstAddr *net.TCPAddr
		if a, ok := src.RemoteAddr().(*net.TCPAddr); ok {
			srcAddr = a
		}
		if a, ok := src.LocalAddr().(*net.TCPAddr); ok {
			dstAddr = a
		}

		if srcAddr == nil || dstAddr == nil {
			_, err := io.WriteString(w, "PROXY UNKNOWN\r\n")
			return err
		}

		family := "TCP4"
		if srcAddr.IP.To4() == nil {
			family = "TCP6"
		}
		_, err := fmt.Fprintf(w, "PROXY %s %s %d %s %d\r\n", family, srcAddr.IP, srcAddr.Port, dstAddr.IP, dstAddr.Port)
		return err
	default:
		return fmt.Errorf("PROXY protocol version %d not supported", dp.ProxyProtocolVersion)
	}
}

// proxyCopy is the function that copies bytes around.
// It's a named function instead of a func literal so users get
// named goroutines in debug goroutine stack dumps.
func proxyCopy(errc chan<- error, dst, src net.Conn) {
	// Before we unwrap src and/or dst, copy any buffered data.
	if wc, ok := src.(*Conn); ok && len(wc.Peeked) > 0 {
		if _, err := dst.Write(wc.Peeked); err != nil {
			errc <- err
			return
		}
		wc.Peeked = nil
	}

	// Unwrap the src and dst from *Conn to *net.TCPConn so Go
	// 1.11's splice optimization kicks in.
	src = UnderlyingConn(src)
	dst = UnderlyingConn(dst)

	_, err := io.Copy(dst, src)
	errc <- err
}

func (dp *DialProxy) keepAlivePeriod() time.Duration {
	if dp.KeepAlivePeriod != 0 {
		return dp.KeepAlivePeriod
	}
	return time.Minute
}

func (dp *DialProxy) dialTimeout() time.Duration {
	if dp.DialTimeout > 0 {
		return dp.DialTimeout
	}
	return 10 * time.Second
}

var defaultDialer = new(net.Dialer)

func (dp *DialProxy) dialContext() func(ctx context.Context, network, address string) (net.Conn, error) {
	if dp.DialContext != nil {
		return dp.DialContext
	}
	return defaultDialer.DialContext
}

func (dp *DialProxy) onDialError() func(src net.Conn, dstDialErr error) {
	if dp.OnDialError != nil {
		return dp.OnDialError
	}
	return func(src net.Conn, dstDialErr error) {
		zlg.Error(dstDialErr, "Trouble dialing upstream",
			zap.String("incoming", src.RemoteAddr().String()),
			zap.String("upstream", dp.Addr),
		)
		src.Close()
	}
}
