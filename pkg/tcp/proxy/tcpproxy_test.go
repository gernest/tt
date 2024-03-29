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
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	proxyPkg "github.com/gernest/tt/pkg/proxy"
	"github.com/gernest/tt/pkg/tcp/middlewares"
)

type noopTarget struct{}

func (noopTarget) HandleConn(_ context.Context, _ net.Conn) {}

type neverEnding byte

func (b neverEnding) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = byte(b)
	}
	return len(p), nil
}

type recordWritesConn struct {
	buf bytes.Buffer
	net.Conn
}

func (c *recordWritesConn) Write(p []byte) (int, error) {
	c.buf.Write(p)
	return len(p), nil
}

func (c *recordWritesConn) Read(p []byte) (int, error) { return 0, io.EOF }

func clientHelloRecord(t *testing.T, hostName string) string {
	rec := new(recordWritesConn)
	cl := tls.Client(rec, &tls.Config{ServerName: hostName})
	cl.Handshake()

	s := rec.buf.String()
	if !strings.Contains(s, hostName) {
		t.Fatalf("clientHello sent in test didn't contain %q", hostName)
	}
	return s
}

func TestSNI(t *testing.T) {
	const hostName = "foo.com"
	greeting := clientHelloRecord(t, hostName)
	got := middlewares.ClientHelloServerName(bufio.NewReader(strings.NewReader(greeting)))
	if got != hostName {
		t.Errorf("got SNI %q; want %q", got, hostName)
	}
}

func TestProxyStartNone(t *testing.T) {
	var p Proxy
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}
}

func newLocalListener(t *testing.T) net.Listener {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		ln, err = net.Listen("tcp", "[::1]:0")
		if err != nil {
			t.Fatal(err)
		}
	}
	return ln
}

const testFrontAddr = "1.2.3.4:567"

func testListenFunc(t *testing.T, ln net.Listener) func(network, laddr string) (net.Listener, error) {
	return func(network, laddr string) (net.Listener, error) {
		if network != "tcp" {
			t.Errorf("got Listen call with network %q, not tcp", network)
			return nil, errors.New("invalid network")
		}
		if laddr != testFrontAddr {
			t.Fatalf("got Listen call with laddr %q, want %q", laddr, testFrontAddr)
			panic("bogus address")
		}
		return ln, nil
	}
}

func testProxy(t *testing.T, front net.Listener) (*Proxy, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Proxy{
		configMap:  make(configMap),
		ListenFunc: testListenFunc(t, front),
		ctx:        ctx,
		opts:       &proxyPkg.Options{},
	}
	p.opts.AllowedPorts = []int{567}
	return p, cancel
}

func TestProxyAlwaysMatch(t *testing.T) {
	front := newLocalListener(t)
	defer front.Close()
	back := newLocalListener(t)
	defer back.Close()

	p, cancel := testProxy(t, front)
	defer cancel()
	p.AddRoute(testFrontAddr, To(back.Addr().String()))
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}

	toFront, err := net.Dial("tcp", front.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer toFront.Close()

	fromProxy, err := back.Accept()
	if err != nil {
		t.Fatal(err)
	}
	const msg = "message"
	io.WriteString(toFront, msg)

	buf := make([]byte, len(msg))
	if _, err := io.ReadFull(fromProxy, buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != msg {
		t.Fatalf("got %q; want %q", buf, msg)
	}
}

func TestProxySNI(t *testing.T) {
	front := newLocalListener(t)
	defer front.Close()

	backFoo := newLocalListener(t)
	defer backFoo.Close()
	backBar := newLocalListener(t)
	defer backBar.Close()

	p, cancel := testProxy(t, front)
	defer cancel()
	p.AddSNIRoute(testFrontAddr, "foo.com", To(backFoo.Addr().String()))
	p.AddSNIRoute(testFrontAddr, "bar.com", To(backBar.Addr().String()))
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}

	toFront, err := net.Dial("tcp", front.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer toFront.Close()

	msg := clientHelloRecord(t, "bar.com")
	io.WriteString(toFront, msg)

	fromProxy, err := backBar.Accept()
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, len(msg))
	if _, err := io.ReadFull(fromProxy, buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != msg {
		t.Fatalf("got %q; want %q", buf, msg)
	}
}

func TestProxyPROXYOut(t *testing.T) {
	front := newLocalListener(t)
	defer front.Close()
	back := newLocalListener(t)
	defer back.Close()

	p, cancel := testProxy(t, front)
	defer cancel()
	p.AddRoute(testFrontAddr, &DialProxy{
		Addr:                 back.Addr().String(),
		ProxyProtocolVersion: 1,
	})
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}

	toFront, err := net.Dial("tcp", front.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	io.WriteString(toFront, "foo")
	toFront.Close()

	fromProxy, err := back.Accept()
	if err != nil {
		t.Fatal(err)
	}

	bs, err := ioutil.ReadAll(fromProxy)
	if err != nil {
		t.Fatal(err)
	}

	want := fmt.Sprintf("PROXY TCP4 %s %d %s %d\r\nfoo", toFront.LocalAddr().(*net.TCPAddr).IP, toFront.LocalAddr().(*net.TCPAddr).Port, toFront.RemoteAddr().(*net.TCPAddr).IP, toFront.RemoteAddr().(*net.TCPAddr).Port)
	if string(bs) != want {
		t.Fatalf("got %q; want %q", bs, want)
	}
}

type tlsServer struct {
	Listener net.Listener
	Domain   string
	Test     *testing.T
}

func (t *tlsServer) Start() {
	cert, acmeCert := cert(t.Test, t.Domain), cert(t.Test, t.Domain+".acme.invalid")
	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert, acmeCert},
	}
	cfg.BuildNameToCertificate()

	go func() {
		for {
			rawConn, err := t.Listener.Accept()
			if err != nil {
				return // assume Close()
			}

			conn := tls.Server(rawConn, cfg)
			if _, err = io.WriteString(conn, t.Domain); err != nil {
				t.Test.Errorf("writing to tlsconn: %s", err)
			}
			conn.Close()
		}
	}()
}

func (t *tlsServer) Close() {
	t.Listener.Close()
}

// cert creates a well-formed, but completely insecure self-signed
// cert for domain.
func cert(t *testing.T, domain string) tls.Certificate {
	private, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Co"},
		},
		DNSNames:              []string{domain},
		NotBefore:             time.Time{},
		NotAfter:              time.Now().Add(60 * time.Minute),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &private.PublicKey, private)
	if err != nil {
		t.Fatal(err)
	}

	var cert, key bytes.Buffer
	pem.Encode(&cert, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	pem.Encode(&key, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(private)})

	tlscert, err := tls.X509KeyPair(cert.Bytes(), key.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	return tlscert
}

// newTLSServer starts a TLS server that serves a self-signed cert for
// domain, and a corresonding acme.invalid dummy domain.
func newTLSServer(t *testing.T, domain string) net.Listener {
	cert, acmeCert := cert(t, domain), cert(t, domain+".acme.invalid")

	l := newLocalListener(t)
	go func() {
		for {
			rawConn, err := l.Accept()
			if err != nil {
				return // assume closed
			}

			cfg := &tls.Config{
				Certificates: []tls.Certificate{cert, acmeCert},
			}
			cfg.BuildNameToCertificate()
			conn := tls.Server(rawConn, cfg)
			if _, err = io.WriteString(conn, domain); err != nil {
				t.Errorf("writing to tlsconn: %s", err)
			}
			conn.Close()
		}
	}()

	return l
}

func readTLS(dest, domain string) (string, error) {
	conn, err := tls.Dial("tcp", dest, &tls.Config{
		ServerName:         domain,
		InsecureSkipVerify: true,
	})
	if err != nil {
		return "", err
	}
	defer conn.Close()

	bs, err := ioutil.ReadAll(conn)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

func TestProxyACME(t *testing.T) {

	front := newLocalListener(t)
	defer front.Close()

	backFoo := newTLSServer(t, "foo.com")
	defer backFoo.Close()
	backBar := newTLSServer(t, "bar.com")
	defer backBar.Close()
	backQuux := newTLSServer(t, "quux.com")
	defer backQuux.Close()

	p, cancel := testProxy(t, front)
	defer cancel()
	p.AddAllowACMESearch(testFrontAddr)
	p.AddSNIRoute(testFrontAddr, "foo.com", To(backFoo.Addr().String()))
	p.AddSNIRoute(testFrontAddr, "bar.com", To(backBar.Addr().String()))
	p.AddStopACMESearch(testFrontAddr)
	p.AddSNIRoute(testFrontAddr, "quux.com", To(backQuux.Addr().String()))
	if err := p.Start(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		domain, want string
		succeeds     bool
	}{
		{"foo.com", "foo.com", true},
		{"bar.com", "bar.com", true},
		{"quux.com", "quux.com", true},
		{"xyzzy.com", "", false},
		{"foo.com.acme.invalid", "foo.com", true},
		{"bar.com.acme.invalid", "bar.com", true},
		{"quux.com.acme.invalid", "", false},
	}
	for d, test := range tests {
		got, err := readTLS(front.Addr().String(), test.domain)
		if test.succeeds {
			if err != nil {
				t.Fatalf("readTLS %d:%q got error %q, want nil", d, test.domain, err)
			}
			if got != test.want {
				t.Fatalf("readTLS %d:%q got %q, want %q", d, test.domain, got, test.want)
			}
		} else if err == nil {
			t.Fatalf("readTLS %d:%q unexpectedly succeeded", d, test.domain)
		}
	}
}
