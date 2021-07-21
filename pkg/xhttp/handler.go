package xhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/gernest/tt/api"
	accesslog "github.com/gernest/tt/pkg/access_log"
	"github.com/gernest/tt/pkg/hrf"
	"github.com/gernest/tt/pkg/meta"
	"github.com/gernest/tt/pkg/reverse"
	handlerv1 "github.com/gernest/tt/wasm/v1/handler"
	handlerv2 "github.com/gernest/tt/wasm/v2/handler"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
)

func (p *Proxy) Handler(routes []*api.Route) (*mux.Router, error) {
	m := mux.NewRouter()
	for _, route := range routes {
		info := &meta.RouteInfo{
			Route: route,
		}
		h := http.Handler(HNoOp{})
		if route.IsHealthEndpoint {
			h = &HHeath{
				fn: p.Health,
			}
		} else if len(route.LoadBalance) > 0 {
			rh, err := reverse.New(route)
			if err != nil {
				return nil, err
			}
			h = rh
		}
		chain := alice.New(p.buildMiddlewares(route)...).Then(h)
		for _, r := range buildRouters(m, route, info) {
			for _, hn := range route.HostNames {
				r = r.Host(hn)
			}
			r.Handler(&H{
				h:    chain,
				info: info,
			})
		}

		// build middlewares
	}
	return m, nil
}

func buildRouters(m *mux.Router, route *api.Route, info *meta.RouteInfo) (routes []*mux.Route) {
	switch e := route.Rule.Match.(type) {
	case *api.Rule_All:
		r := m.Name(route.Name)
		// We disallow nesting rules more than this depth. All rules here bust be concrete forhttp
		for _, v := range e.All.Rules {
			if h := v.GetHttp(); h != nil {
				r = httpMatch(h, r, info)
			}
		}
		routes = append(routes, r)
	case *api.Rule_Any:
		for i, v := range e.Any.Rules {
			if h := v.GetHttp(); h != nil {
				r := m.Name(fmt.Sprintf("%s-any-%d", route.Name, i))
				r = httpMatch(h, r, info)
				routes = append(routes, r)
			}
		}
	case *api.Rule_Not:
		//TODO: support this?
	case *api.Rule_Http:
		r := m.Name(route.Name)
		r = httpMatch(route.Rule.GetHttp(), r, info)
		routes = append(routes, r)
	}
	return
}

func httpMatch(a *api.Rule_HTTP, route *mux.Route, info *meta.RouteInfo) (r *mux.Route) {
	r = route
	if v := a.GetMethods(); v != nil {
		var methods []string
		for _, x := range v.GetList() {
			methods = append(methods, x.String())
		}
		r = r.Methods(methods...)
	}
	if v := a.GetPath(); v != nil {
		switch v.Type {
		case api.Rule_HTTP_Path_Prefix:
			prefix := v.Value
			if !strings.HasPrefix(prefix, "/") {
				prefix = "/" + prefix
			}
			r = r.PathPrefix(prefix)
		case api.Rule_HTTP_Path_Exact:
			exact := v.Value
			if !strings.HasPrefix(exact, "/") {
				exact = "/" + exact
			}
			r = r.Path(exact)
		case api.Rule_HTTP_Path_RegularExpression:
			regex := v.Value
			if strings.HasPrefix(regex, "/") {
				regex = strings.TrimPrefix(regex, "/")
			}
			r = r.Path(fmt.Sprintf("/{path:%s}", regex))
		}
	}
	if v := a.GetHeaders(); v != nil {
		var plain, regex []string
		for _, x := range v.GetList() {
			switch x.Type {
			case api.Rule_HTTP_KeyValue_Exact:
				if x.Value != "" {
					plain = append(plain, x.Name, x.Value)
				}
			case api.Rule_HTTP_KeyValue_RegularExpression:
				if x.Value != "" {
					regex = append(regex, x.Value)
				}
			}
		}
		if len(plain) > 0 {
			r = r.Headers(plain...)
		}
		if len(regex) > 0 {
			r = r.HeadersRegexp(regex...)
		}
	}
	if v := a.GetQueryParam(); v != nil {
		var all []string
		for _, x := range v.GetList() {
			switch x.Type {
			case api.Rule_HTTP_KeyValue_Exact:
				if x.Value != "" {
					all = append(all, x.Name, x.Value)
				}
			case api.Rule_HTTP_KeyValue_RegularExpression:
				if x.Value != "" {
					all = append(all, x.Value)
				}
			}
		}
		if len(all) > 0 {
			r = r.Queries(all...)
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
	next := d.Get()
	next = accesslog.Instrument(next)
	next.ServeHTTP(w, r)
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

type HNoOp struct {
	r *api.Route
}

func (n *HNoOp) Route() *api.Route { return n.r }

func (HNoOp) ServeHTTP(w http.ResponseWriter, r *http.Request) {}

type HHeath struct {
	fn func() hrf.Health
	r  *api.Route
}

func (h HHeath) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/health+json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(h.fn())
}

type H404 struct{}

func (H404) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

func HealthEndpoint() *api.Route {
	return &api.Route{
		Name:             "helathz",
		Protocol:         api.Protocol_HTTP,
		IsHealthEndpoint: true,
		Rule: &api.Rule{
			Match: &api.Rule_Http{
				Http: &api.Rule_HTTP{
					Match: &api.Rule_HTTP_Path_{
						Path: &api.Rule_HTTP_Path{
							Value: "/healthz",
						},
					},
				},
			},
		},
	}
}

type H struct {
	h    http.Handler
	info *meta.RouteInfo
}

func (h *H) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m := meta.GetMetics(r.Context()); m != nil {
		// we are updating analytics details since this is the last handler in the
		// chain. Target is updated by the reverse proxy handler.
		m.Info.Route = h.info.Route.Name
		m.Info.Service = h.info.Route.Service
		m.Info.VirtualHosts = h.info.Route.HostNames
	}
	h.h.ServeHTTP(w, r)
}

func StripPathPrefix(mw *api.Middleware_StripPathPrefix) alice.Constructor {
	return func(h http.Handler) http.Handler {
		return http.StripPrefix(mw.Prefix, h)
	}
}

func (p *Proxy) buildMiddlewares(r *api.Route) (mw []alice.Constructor) {
	for _, w := range r.GetMiddlewares().GetList() {
		if h := p.ware(w); h != nil {
			mw = append(mw, h)
		}
	}
	return
}

func (p *Proxy) ware(mw *api.Middleware) alice.Constructor {
	if strip := mw.GetStripPathPrefix(); strip != nil {
		return StripPathPrefix(strip)
	}
	if ws := mw.GetWasm(); ws != nil {
		switch ws.Version {
		case api.Middleware_V1:
			m, err := handlerv1.New(p.ctx, p.opts.Wasm.Dir, ws)
			if err != nil {
				return nil
			}
			return m.Handle
		case api.Middleware_V2:
			m, err := handlerv2.New(p.ctx, p.opts.Wasm.Dir, ws)
			if err != nil {
				return nil
			}
			return m.Handle
		}
	}
	return nil
}
