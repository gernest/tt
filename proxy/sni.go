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
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	"strings"

	"github.com/gernest/tt/pkg/tcp"
	"github.com/gernest/tt/pkg/tcp/dtls"
	"github.com/gernest/tt/zlg"
	"go.uber.org/zap"
)

type sniMatch struct {
	matcher Matcher
	target  tcp.Target
}

var _ tcp.Route = (*sniMatch)(nil)

func (m sniMatch) Match(ctx context.Context, br *bufio.Reader) (tcp.Target, string) {
	sni := clientHelloServerName(br)
	zlg.Debug("read sni", zap.String("sni", sni), zap.String("component", "sni_match"))
	if m.matcher(ctx, sni) {
		zlg.Debug("sni matched", zap.String("sni", sni), zap.String("component", "sni_match"))
		meta := tcp.GetContextMeta(ctx)
		meta.ServerName.Store(sni)
		return m.target, sni
	}
	return nil, ""
}

// acmeMatch matches "*.acme.invalid" ACME tls-sni-01 challenges and
// searches for a Target in cfg.acmeTargets that has the challenge
// response.
type acmeMatch struct {
	cfg *config
}

var _ tcp.Route = (*acmeMatch)(nil)

func (m *acmeMatch) Match(ctx context.Context, br *bufio.Reader) (tcp.Target, string) {
	sni := clientHelloServerName(br)
	if !strings.HasSuffix(sni, ".acme.invalid") {
		return nil, ""
	}
	meta := tcp.GetContextMeta(ctx)
	meta.ACME.Store(true)

	// TODO: cache. ACME issuers will hit multiple times in a short
	// burst for each issuance event. A short TTL cache + singleflight
	// should have an excellent hit rate.
	// TODO: maybe an acme-specific timeout as well?
	// TODO: plumb context upwards?
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := make(chan tcp.Target, len(m.cfg.acmeTargets))
	for _, target := range m.cfg.acmeTargets {
		go tryACME(ctx, ch, target, sni)
	}
	for range m.cfg.acmeTargets {
		if target := <-ch; target != nil {
			return target, sni
		}
	}

	// No target was happy with the provided challenge.
	return nil, ""
}

func tryACME(ctx context.Context, ch chan<- tcp.Target, dest tcp.Target, sni string) {
	var ret tcp.Target
	defer func() { ch <- ret }()

	conn, targetConn := net.Pipe()
	defer conn.Close()
	go dest.HandleConn(ctx, targetConn)

	deadline, ok := ctx.Deadline()
	if ok {
		conn.SetDeadline(deadline)
	}

	client := tls.Client(conn, &tls.Config{
		ServerName:         sni,
		InsecureSkipVerify: true,
	})
	if err := client.Handshake(); err != nil {
		// TODO: log?
		return
	}
	certs := client.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		// TODO: log?
		return
	}
	// acme says the first cert offered by the server must match the
	// challenge hostname.
	if err := certs[0].VerifyHostname(sni); err != nil {
		// TODO: log?
		return
	}

	// Target presented what looks like a valid challenge
	// response, send it back to the matcher.
	ret = dest
}

// clientHelloServerName returns the SNI server name inside the TLS ClientHello,
// without consuming any bytes from br.
// On any error, the empty string is returned.
func clientHelloServerName(br *bufio.Reader) (sni string) {
	const recordHeaderLen = 5
	hdr, err := br.Peek(recordHeaderLen)
	if err != nil {
		return ""
	}
	const recordTypeHandshake = 0x16
	if hdr[0] != recordTypeHandshake {
		return ""
	}
	recLen := int(hdr[3])<<8 | int(hdr[4]) // ignoring version in hdr[1:3]
	helloBytes, err := br.Peek(recordHeaderLen + recLen)
	if err != nil {
		return ""
	}
	tls.Server(sniSniffConn{r: bytes.NewReader(helloBytes)}, &tls.Config{
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			sni = hello.ServerName
			return nil, nil
		},
	}).Handshake()
	if sni == "" {
		// Not TLS try dtls
		return dtls.ClientHelloServerNameDTLS(br)
	}
	return
}

// sniSniffConn is a net.Conn that reads from r, fails on Writes,
// and crashes otherwise.
type sniSniffConn struct {
	r        io.Reader
	net.Conn // nil; crash on any unexpected use
}

func (c sniSniffConn) Read(p []byte) (int, error) { return c.r.Read(p) }
func (sniSniffConn) Write(p []byte) (int, error)  { return 0, io.EOF }
