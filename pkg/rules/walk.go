package rules

import (
	"errors"

	"github.com/gernest/tt/api"
)

var SkiList = errors.New("rule:Slip")

type WalkFunc func(rule *api.Rule, isList bool) error

func Walk(r *api.Rule, fn WalkFunc) error {
	if r == nil {
		return nil
	}
	switch e := r.Match.(type) {
	case *api.Rule_All:
		if err := fn(r, true); err != nil {
			if errors.Is(err, SkiList) {
				return nil
			}
			return err
		}
		for _, v := range e.All.Rules {
			if err := Walk(v, fn); err != nil {
				return err
			}
		}
		return nil
	case *api.Rule_Any:
		if err := fn(r, true); err != nil {
			if errors.Is(err, SkiList) {
				return nil
			}
			return err
		}
		for _, v := range e.Any.Rules {
			if err := Walk(v, fn); err != nil {
				return err
			}
		}
		return nil
	case *api.Rule_Not:
		if err := fn(r, true); err != nil {
			if errors.Is(err, SkiList) {
				return nil
			}
			return err
		}
		if err := Walk(e.Not, fn); err != nil {
			return err
		}
		return nil
	}
	return fn(r, false)
}
