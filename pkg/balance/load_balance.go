package balance

import (
	"errors"
	"net/url"

	"github.com/gernest/tt/api"
	"github.com/smallnest/weighted"
)

var ErrConfigNotFound = errors.New("balance: Missing load balancer configuration")

type Target struct {
	URL *url.URL
}

type Balance interface {
	Next() *Target
}

type balanceFn func() *Target

func (fn balanceFn) Next() *Target {
	return fn()
}

func target(w *api.WeightedAddr) (*Target, error) {
	u, err := url.Parse(w.Addr.Address)
	if err != nil {
		return nil, err
	}
	return &Target{URL: u}, nil
}

func FromRoute(r *api.Route) (Balance, error) {
	return FromWeightedAddr(r.LoadBalanceAlgo, r.LoadBalance...)
}

func FromWeightedAddr(algo api.Route_LoadBalanceAlgo, targets ...*api.WeightedAddr) (Balance, error) {
	if len(targets) == 1 {
		t, err := target(targets[0])
		if err != nil {
			return nil, err
		}
		return balanceFn(func() *Target {
			return t
		}), nil
	}
	switch algo {
	case api.Route_RoundRobinWeighted:
		w := &weighted.RRW{}
		for _, v := range targets {
			t, err := target(v)
			if err != nil {
				return nil, err
			}
			w.Add(t, int(v.Weight))
		}
		return balanceFn(func() *Target {
			return w.Next().(*Target)
		}), nil
	case api.Route_SmoothWeighted:
		w := &weighted.SW{}
		for _, v := range targets {
			t, err := target(v)
			if err != nil {
				return nil, err
			}
			w.Add(t, int(v.Weight))
		}
		return balanceFn(func() *Target {
			return w.Next().(*Target)
		}), nil

	case api.Route_RandomWeighted:
		w := &weighted.RandW{}
		for _, v := range targets {
			t, err := target(v)
			if err != nil {
				return nil, err
			}
			w.Add(t, int(v.Weight))
		}
		return balanceFn(func() *Target {
			return w.Next().(*Target)
		}), nil
	}
	return nil, ErrConfigNotFound
}
