package proxy

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"

	"github.com/gernest/tt/pkg/tcp"
)

var _ net.Conn = *&sniSniffConn{}

func TestFancyCopy(t *testing.T) {
	t.Run("Bidirectional", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.TODO(), 100*time.Millisecond)
		defer cancel()

		// we test that what is read upstream is actually written downstream and what
		// is read downstream is written upstream
		ctx = tcp.UpdateContext(ctx, func(cm *tcp.ContextMeta) {})
		ping := []byte("ping")
		pong := []byte("pong")
		var up, down bytes.Buffer
		downstream := &conn{
			readFn: func(b []byte) (int, error) {
				return copy(b, ping), nil
			},
			writeFn: func(b []byte) (int, error) {
				return down.Write(b)
			},
		}
		upstream := &conn{
			writeFn: func(b []byte) (n int, err error) {
				return up.Write(b)
			},
			readFn: func(b []byte) (int, error) {
				return copy(b, pong), nil
			},
		}
		go func() {
			Copy(ctx, downstream, upstream)
		}()
		<-ctx.Done()
		if !bytes.Contains(up.Bytes(), ping) {
			t.Error("expected ping to be copied upstream")
		}
		if !bytes.Contains(down.Bytes(), pong) {
			t.Error("expected pong to be copied downstream")
		}
	})
}

type conn struct {
	readFn, writeFn func([]byte) (int, error)
	net.Conn
}

func (c *conn) Read(b []byte) (n int, err error) {
	return c.readFn(b)
}

func (c *conn) Write(b []byte) (n int, err error) {
	return c.writeFn(b)
}
