package proxy

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"

	"golang.org/x/net/bpf"
)

// BufferSize this is the size of data that is read/written by default.
const BufferSize = KiB

type transmitOptions struct {
	filter Filter
	take   func(*Meta) error
}

type Meta struct {
	Stat Stat
}

type Stat struct {
	Read  int64
	Write int64
}

// Filter provides bpf filters for passing tcp traffick
type Filter struct {
	Init   *bpf.VM
	Before *bpf.VM
	End    *bpf.VM
}

type transit struct {
	src      net.Conn
	offset   int64
	dest     net.Conn
	opts     transmitOptions
	meta     Meta
	throttle struct {
		read, write Throttle
	}
	buf *bytes.Buffer
}

func (s *transit) take() error {
	if s.opts.take != nil {
		return s.opts.take(&s.meta)
	}
	return nil
}

func (s *transit) copy(ctx context.Context) error {
	// start by initial read
	if err := s.read(ctx); err != nil {
		return err
	}
	if err := s.init(); err != nil {
		return err
	}
	for {
		if ctx.Err() != nil {
			return nil
		}
		if err := s.take(); err != nil {
			return nil
		}
		err := s.read(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				if err := s.terminateConn(); err != nil {
					return err
				}
				return s.write(ctx)
			}
			return err
		}
		if err = s.accept(); err != nil {
			return err
		}
		if err = s.write(ctx); err != nil {
			return err
		}
	}
}

func (s *transit) read(ctx context.Context) error {
	if !s.throttle.read.Take() {
		// we can wait until we are allowed to read again
		if err := s.throttle.read.Wait(ctx); err != nil {
			return err
		}
	}
	s.buf.Reset()
	if wc, ok := s.src.(*Conn); ok && len(wc.Peeked) > 0 {
		if _, err := s.buf.Write(wc.Peeked); err != nil {
			return err
		}
		wc.Peeked = nil
	}
	n, err := io.CopyN(s.buf, s.src, BufferSize)
	if err != nil {
		return err
	}
	s.meta.Stat.Read += n
	// we default to write everything that we read
	s.offset = n
	return nil
}

func (s *transit) write(ctx context.Context) error {
	if !s.throttle.write.Take() {
		// we can wait until we are allowed to read again
		if err := s.throttle.write.Wait(ctx); err != nil {
			return err
		}
	}
	if w := s.offset; w != 0 {
		if w < int64(s.buf.Len()) {
			n, err := s.dest.Write(s.buf.Bytes()[:int(w)])
			s.meta.Stat.Write += int64(n)
			return err
		}
		n, err := s.buf.WriteTo(s.dest)
		if err != nil {
			return err
		}
		s.meta.Stat.Write += n
	}
	return nil
}

func (s *transit) init() error {
	return s.validate(s.opts.filter.Init)
}

func (s *transit) validate(v *bpf.VM) error {
	if v != nil {
		n, err := v.Run(s.buf.Bytes())
		if err != nil {
			return err
		}
		if n != -1 {
			s.offset = int64(n)
		} else {
			s.offset = int64(s.buf.Len())
		}
	}
	return nil
}

func (s *transit) terminateConn() error {
	return s.validate(s.opts.filter.End)
}

func (s *transit) accept() error {
	return s.validate(s.opts.filter.Before)
}

// Copy starts two goroutines. On that copy from=>to and another that copies
// to>from
func Copy(ctx context.Context, from, to net.Conn) error {
	bctx, cancel := context.WithCancel(ctx)
	// request => upstream
	a := transit{
		src:  from,
		dest: to,
		buf:  new(bytes.Buffer),
	}
	go func() {
		defer cancel()
		if err := a.copy(bctx); err != nil {
			// do something
		}
	}()
	// upstream => request
	b := transit{
		src:  to,
		dest: from,
		buf:  new(bytes.Buffer),
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
