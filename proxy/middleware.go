package proxy

import "github.com/gernest/tt/api"

type middleareFunc func(Target) Target

type chain []middleareFunc

func (c chain) then(t Target) Target {
	for _, h := range c {
		t = h(t)
	}
	return t
}

func buildMiddleares(r *api.Route) chain {
	return chain{}
}
