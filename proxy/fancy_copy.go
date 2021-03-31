package proxy

import (
	"context"
	"errors"
	"io"
	"net"

	"github.com/gernest/tt/proxy/buffer"
	"github.com/valyala/bytebufferpool"
	"golang.org/x/time/rate"
)

// BufferSize this is the size of data that is read/written by default.
const BufferSize = KiB

type limit interface {
	WaitN(context.Context, int) error
}

type noLimit struct{}

func (noLimit) WaitN(context.Context, int) error { return nil }

func newRate(v float64) limit {
	if v == 0 {
		return noLimit{}
	}
	return rate.NewLimiter(rate.Limit(v), 0)
}

type transit struct {
	src             net.Conn
	dest            net.Conn
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
	if wc, ok := s.src.(*Conn); ok && len(wc.Peeked) > 0 {
		if _, err := s.buf.Write(wc.Peeked); err != nil {
			return err
		}
		wc.Peeked = nil
	}
	if err := s.rate.WaitN(ctx, s.buf.Len()); err != nil {
		return err
	}
	n, err := io.CopyN(s.buf, s.src, BufferSize)
	if err != nil {
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
		n, err := s.buf.WriteTo(s.dest)
		if err != nil {
			return err
		}
		s.recordWrite(n)
	}
	return nil
}

func (s *transit) recordWrite(n int64) {
	if s.onWrite != nil {
		s.onWrite(s.meta, n)
	}
}

// Copy starts two goroutines. On that copy from=>to and another that copies
// to>from
// from is downstream connection while to is the upstream connection.
func Copy(ctx context.Context, from, to net.Conn) error {
	var meta *ContextMeta
	CheckContext(ctx, func(cm *ContextMeta) {
		meta = cm
	})
	bctx, cancel := context.WithCancel(ctx)
	down := buffer.Get()
	defer buffer.Put(down)
	up := buffer.Get()
	defer buffer.Put(up)
	downstream := transit{
		src:  from,
		dest: to,
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
		src:  to,
		dest: from,
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
	select {
	case <-bctx.Done():
		return bctx.Err()
	default:
		return nil
	}
}
