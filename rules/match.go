package rules

import (
	"context"

	"github.com/gernest/tt/api"
)

type MatchFunc func(ctx context.Context, meta *api.Context) bool

func (fn MatchFunc) Match(ctx context.Context, meta *api.Context) bool {
	return fn(ctx, meta)
}

var noop = MatchFunc(func(ctx context.Context, meta *api.Context) bool { return true })

func and(a, b Matcher) Matcher {
	return MatchFunc(func(ctx context.Context, meta *api.Context) bool {
		return a.Match(ctx, meta) && b.Match(ctx, meta)
	})
}

func or(a, b Matcher) Matcher {
	return MatchFunc(func(ctx context.Context, meta *api.Context) bool {
		return a.Match(ctx, meta) || b.Match(ctx, meta)
	})
}

func not(a Matcher) Matcher {
	return MatchFunc(func(ctx context.Context, meta *api.Context) bool {
		return !a.Match(ctx, meta)
	})
}

func build(r *api.Rule) (match Matcher, err error) {
	switch e := r.Match.(type) {
	case *api.Rule_All:
		for _, v := range e.All.Rules {
			n, err := build(v)
			if err != nil {
				return nil, err
			}
			match = and(match, n)
		}
	case *api.Rule_Any:
		for _, v := range e.Any.Rules {
			n, err := build(v)
			if err != nil {
				return nil, err
			}
			match = or(match, n)
		}
	case *api.Rule_Not:
		n, err := build(r)
		if err != nil {
			return nil, err
		}
		match = not(n)
	case *api.Rule_Tcp:
		return buildTCP(e)
	case *api.Rule_Http:
		return buildHTTP(e)
	}
	return
}

func buildTCP(r *api.Rule_Tcp) (match Matcher, err error) {
	switch e := r.Tcp.Match.(type) {
	case *api.Rule_TCP_Port:
		match = MatchFunc(func(ctx context.Context, meta *api.Context) bool {
			return e.Port == uint32(meta.Info.ListenPort)
		})
	case *api.Rule_TCP_Ports:
		match = noop
	case *api.Rule_TCP_Sni:
		match = MatchFunc(func(ctx context.Context, meta *api.Context) bool {
			return e.Sni == meta.Info.Sni.GetValue()
		})
	}
	return nil, nil
}

func buildHTTP(r *api.Rule_Http) (match Matcher, err error) {
	switch e := r.Http.Match.(type) {
	case *api.Rule_HTTP_Host:
		match = MatchFunc(func(ctx context.Context, meta *api.Context) bool {
			return e.Host == meta.Info.Host.GetValue()
		})
	default:
		match = noop
	}
	return nil, nil
}
