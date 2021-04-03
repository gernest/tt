package proxy

import (
	"context"
	"errors"
	"io"
	"net"

	"github.com/gernest/tt/proxy/buffer"
	"github.com/valyala/bytebufferpool"
)

// BufferSize this is the size of data that is read/written by default.
const BufferSize = KiB

type transitConn interface {
	ReadTo(dest io.Writer, size int64) (int64, error)
	io.Writer
}

type Upstream struct {
	copy *connCopy
}

func newUpstream(src, dest net.Conn) *Upstream {
	return &Upstream{copy: &connCopy{src: src, dest: dest}}
}

func (u *Upstream) Write(b []byte) (int, error) {
	return u.copy.write(b)
}

func (u *Upstream) ReadTo(dest io.Writer, size int64) (int64, error) {
	return u.copy.read(dest, size)
}

type Downstream struct {
	copy *connCopy
}

func newDownstream(src, dest net.Conn) *Downstream {
	return &Downstream{copy: &connCopy{src: src, dest: dest}}
}

func (u *Downstream) Write(b []byte) (int, error) {
	return u.copy.write(b)
}

func (u *Downstream) ReadTo(dest io.Writer, size int64) (int64, error) {
	return u.copy.read(dest, size)
}

type transit struct {
	conn            transitConn
	rate            limit
	buf             *bytebufferpool.ByteBuffer
	meta            *ContextMeta
	onRead, onWrite func(*ContextMeta, int64)
}

func (s *transit) copy(ctx context.Context) error {
	for {
		if ctx.Err() != nil {
			return nil
		}
		err := s.read(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return s.write(ctx)
			}
			return err
		}
		if err = s.write(ctx); err != nil {
			return err
		}
	}
}

func (s *transit) read(ctx context.Context) error {
	s.buf.Reset()
	n, err := s.conn.ReadTo(s.buf, BufferSize)
	if err != nil {
		return err
	}
	if err := s.rate.WaitN(ctx, s.buf.Len()); err != nil {
		return err
	}
	s.recordRead(n)
	return nil
}

func (s *transit) recordRead(n int64) {
	if s.onRead != nil {
		s.onRead(s.meta, n)
	}
}

func (s *transit) write(ctx context.Context) error {
	if s.buf.Len() > 0 {
		n, err := s.conn.Write(s.buf.Bytes())
		if err != nil {
			return err
		}
		s.recordWrite(int64(n))
	}
	return nil
}

func (s *transit) recordWrite(n int64) {
	if s.onWrite != nil {
		s.onWrite(s.meta, n)
	}
}

type connCopy struct {
	src  net.Conn
	dest net.Conn
}

func (c connCopy) read(dest io.Writer, size int64) (int64, error) {
	if wc, ok := c.src.(*Conn); ok && len(wc.Peeked) > 0 {
		if n, err := dest.Write(wc.Peeked); err != nil {
			return int64(n), err
		}
		wc.Peeked = nil
	}
	return io.CopyN(dest, c.src, size)
}

func (c connCopy) write(b []byte) (int, error) {
	return c.dest.Write(b)
}

// Copy starts two goroutines. On that copy from=>to and another that copies
// to>from
// from is downstream connection while to is the upstream connection.
func Copy(ctx context.Context, from, to net.Conn) error {
	meta := GetContextMeta(ctx)
	bctx, cancel := context.WithCancel(ctx)
	down := buffer.Get()
	defer buffer.Put(down)
	up := buffer.Get()
	defer buffer.Put(up)
	downstream := transit{
		conn: newDownstream(from, to),
		meta: meta,
		rate: newRate(meta.Speed.Downstream.Load()),
		buf:  down,
		onRead: func(cm *ContextMeta, i int64) {
			// We are reading from downstream
			cm.D.R.Add(i)
		},
		onWrite: func(cm *ContextMeta, i int64) {
			//we are writing to upstream
			cm.U.W.Add(i)
		},
	}
	go func() {
		defer cancel()
		if err := downstream.copy(bctx); err != nil {
			// do something
		}
	}()
	upstream := transit{
		conn: newUpstream(to, from),
		meta: meta,
		rate: newRate(meta.Speed.Upstream.Load()),
		buf:  up,
		onRead: func(cm *ContextMeta, i int64) {
			// we are reading from upstream
			cm.U.R.Add(i)
		},
		onWrite: func(cm *ContextMeta, i int64) {
			//we are writing to downstream
			cm.D.W.Add(i)
		},
	}
	go func() {
		defer cancel()
		if err := upstream.copy(bctx); err != nil {
			// do something
		}
	}()
	<-bctx.Done()
	return ctx.Err()
}
