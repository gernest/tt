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

type transit struct {
	src             net.Conn
	dest            net.Conn
	readRate        *rate.Limiter
	writeRate       *rate.Limiter
	buf             *bytebufferpool.ByteBuffer
	onRead, onWrite func(int64)
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
	if err := s.readRate.WaitN(ctx, s.buf.Len()); err != nil {
		return err
	}
	n, err := io.CopyN(s.buf, s.src, BufferSize)
	if err != nil {
		return err
	}
	s.onRead(n)
	return nil
}

func (s *transit) write(ctx context.Context) error {
	if s.buf.Len() > 0 {
		if err := s.writeRate.WaitN(ctx, s.buf.Len()); err != nil {
			return err
		}
		n, err := s.buf.WriteTo(s.dest)
		if err != nil {
			return err
		}
		s.onWrite(n)
	}
	return nil
}

// Copy starts two goroutines. On that copy from=>to and another that copies
// to>from
// from is downstream connection while to is the upstream connection.
func Copy(ctx context.Context, from, to net.Conn) error {
	bctx, cancel := context.WithCancel(ctx)
	down := buffer.Get()
	defer buffer.Put(down)
	up := buffer.Get()
	defer buffer.Put(up)
	a := transit{
		src:  from,
		dest: to,
		buf:  down,
	}
	go func() {
		defer cancel()
		if err := a.copy(bctx); err != nil {
			// do something
		}
	}()
	b := transit{
		src:  to,
		dest: from,
		buf:  up,
	}
	go func() {
		defer cancel()
		if err := b.copy(bctx); err != nil {
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
