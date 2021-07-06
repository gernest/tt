package proxy

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/gernest/tt/pkg/unit"
	"golang.org/x/time/rate"
)

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

type RateCopy struct {
	WaitN func(int) error
	net.Conn
	OnWrite func(int)
	OnRead  func(int)
}

func (r *RateCopy) Write(b []byte) (n int, err error) {
	r.OnRead(len(b))
	if err := r.WaitN(len(b)); err != nil {
		return 0, err
	}
	defer func() {
		r.OnWrite(n)
	}()
	n, err = r.Conn.Write(b)
	return
}

// Speed is a unit representing amount of bytes per duration
// eg 120kib/s
type Speed string

func (s Speed) Limit() (float64, error) {
	if s == "" {
		return 0, nil
	}
	x := strings.Split(string(s), "/")
	v, err := unit.RAMInBytes(x[0])
	if err != nil {
		return 0, err
	}
	per := time.Second
	if len(x) == 2 {
		switch x[1] {
		case "m":
			per = time.Minute
		case "h":
			per = time.Hour
		}
	}
	return float64(v) / per.Seconds(), nil
}
