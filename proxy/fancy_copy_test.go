package proxy

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"
)

var _ net.Conn = *&sniSniffConn{}

func TestFancyCopy(t *testing.T) {
	t.Run("Bidirectional", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.TODO())
		defer cancel()

		// we test that what is read upstream is actually written downstream and what
		// is read downstream is written upstream
		ctx = UpdateContext(ctx, func(cm *ContextMeta) {})
		ping := []byte("ping")
		pong := []byte("pong")
		var up, down bytes.Buffer
		ok := make(chan struct{})
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
				n, err = up.Write(b)
				ok <- struct{}{}
				return
			},
			readFn: func(b []byte) (int, error) {
				<-ok
				return copy(b, pong), nil
			},
		}
		go func() {
			Copy(ctx, downstream, upstream)
		}()
		time.Sleep(10 * time.Millisecond)
		cancel()
		<-ctx.Done()
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
