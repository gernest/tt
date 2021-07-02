package dtls

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/pion/dtls/v2"
	"github.com/pion/dtls/v2/examples/util"
	"github.com/pion/dtls/v2/pkg/crypto/selfsign"
)

const testDTLSServerName = "dtls.test"

func listenSelfSign(ctx context.Context, port int, sni chan string) {
	// Prepare the IP to connect to
	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port}

	// Generate a certificate and private key to secure the connection
	certificate, genErr := selfsign.GenerateSelfSigned()
	util.Check(genErr)
	config := &dtls.Config{
		Certificates:         []tls.Certificate{certificate},
		ExtendedMasterSecret: dtls.RequireExtendedMasterSecret,
		// Create timeout context for accepted connection.
		ConnectContextMaker: func() (context.Context, func()) {
			return context.WithTimeout(ctx, 30*time.Second)
		},
	}
	config.ServerName = "some-server"

	// Connect to a DTLS server
	listener, err := ListenDTLS("udp", addr)
	util.Check(err)
	defer func() {
		util.Check(listener.Close())
	}()

	fmt.Println("Listening")

	go func() {
		for {
			if ctx.Err() != nil {
				return
			}
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			sni <- ClientHelloServerNameDTLS(bufio.NewReader(conn))
			conn.Close()
		}
	}()
	<-ctx.Done()
}

func dialSelfSign(ctx context.Context, port int) {
	// Prepare the IP to connect to
	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port}

	// Generate a certificate and private key to secure the connection
	certificate, genErr := selfsign.GenerateSelfSigned()
	util.Check(genErr)

	config := &dtls.Config{
		Certificates:         []tls.Certificate{certificate},
		InsecureSkipVerify:   true,
		ServerName:           testDTLSServerName,
		ExtendedMasterSecret: dtls.RequireExtendedMasterSecret,
	}

	// Connect to a DTLS server
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	dtlsConn, err := dtls.DialWithContext(ctx, "udp", addr, config)
	if err != nil {
		return
	}
	dtlsConn.Close()
}

func getPort() int {
	ls, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		ls, err = net.Listen("tcp", "127.0.0.1:0")
	}
	defer ls.Close()
	_, port, _ := net.SplitHostPort(ls.Addr().String())
	a, _ := strconv.Atoi(port)
	return a
}

func TestDTLS_clientHelloServerNameDTLS(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	port := getPort()
	sni := make(chan string, 1)
	go listenSelfSign(ctx, port, sni)
	time.Sleep(time.Millisecond)
	go dialSelfSign(ctx, port)
	select {
	case <-ctx.Done():
		t.Error("timedout")
	case e := <-sni:
		cancel()
		if e != testDTLSServerName {
			t.Errorf("expected %q got %q", testDTLSServerName, e)
		}
	}
}
