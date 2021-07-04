package xhttp

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/gernest/tt/api"
	"github.com/gernest/tt/pkg/reverse"
	"github.com/gorilla/mux"
)

func Handler(routes []*api.Route) (*mux.Router, error) {
	m := mux.NewRouter()
	for _, route := range routes {
		h, err := reverse.New(route)
		if err != nil {
			return nil, err
		}
		for _, r := range buildRouters(m, route) {
			r.Handler(h)
		}
	}
	return m, nil
}

func buildRouters(m *mux.Router, route *api.Route) (routes []*mux.Route) {
	switch e := route.Rule.Match.(type) {
	case *api.Rule_All:
		r := m.Name(route.Name)
		// We disallow nesting rules more than this depth. All rules here bust be concrete forhttp
		for _, v := range e.All.Rules {
			if h := v.GetHttp(); h != nil {
				r = httpMatch(h, r)
			}
		}
		routes = append(routes, r)
	case *api.Rule_Any:
		for i, v := range e.Any.Rules {
			if h := v.GetHttp(); h != nil {
				r := m.Name(fmt.Sprintf("%s-any-%d", route.Name, i))
				r = httpMatch(h, r)
				routes = append(routes, r)
			}
		}
	case *api.Rule_Not:
		//TODO: support this?
	case *api.Rule_Http:
		r := m.Name(route.Name)
		r = httpMatch(route.Rule.GetHttp(), r)
		routes = append(routes, r)
	}
	return
}

func httpMatch(a *api.Rule_HTTP, route *mux.Route) (r *mux.Route) {
	r = route
	if v := a.GetMethods(); v != nil {
		var methods []string
		for _, x := range v.Methods {
			methods = append(methods, x.String())
		}
		r = r.Methods(methods...)
	}
	if v := a.GetHost(); v != "" {
		r = r.Host(v)
	}
	if v := a.GetPath(); v != nil {
		if prefix := v.GetPrefix(); prefix != "" {
			r = r.PathPrefix(prefix)
		}
		if exact := v.GetExact(); exact != "" {
			if strings.HasPrefix(exact, "/") {
				exact = "/" + exact
			}
			r = r.Path(exact)
		}
		if regex := v.GetRegexp(); regex != "" {
			if strings.HasPrefix(regex, "/") {
				regex = strings.TrimPrefix(regex, "/")
			}
			r = r.Path(fmt.Sprintf("/{path:%s}", regex))
		}
	}
	if v := a.GetHeaders(); v != nil {
		var plain, regex []string
		for _, x := range v.Headers {
			if exact := x.Value.GetExact(); exact != "" {
				plain = append(plain, x.Key, exact)
			}
			if reg := x.Value.GetRegexp(); reg != "" {
				regex = append(regex, reg)
			}
		}
		if len(plain) > 0 {
			r = r.Headers(plain...)
		}
		if len(regex) > 0 {
			r = r.HeadersRegexp(regex...)
		}
	}
	return
}

var _ http.Handler = (*Dynamic)(nil)

type Dynamic struct {
	Get func() http.Handler
}

func NewDynamic(ctx context.Context, handlerChan <-chan http.Handler, base http.Handler) *Dynamic {
	return &Dynamic{
		Get: ReloadHand(ctx, handlerChan, base),
	}
}

func (d *Dynamic) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.Get().ServeHTTP(w, r)
}

func ReloadHand(ctx context.Context, handlerChan <-chan http.Handler, base http.Handler) func() http.Handler {
	var h atomic.Value
	if base == nil {
		base = &H404{}
	}
	h.Store(base)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case hand := <-handlerChan:
				h.Store(hand)
			}
		}
	}()
	return func() http.Handler {
		return h.Load().(http.Handler)
	}
}

type H404 struct{}

func (H404) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}
