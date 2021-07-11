package xhttp

import (
	"context"
	"net"
	"net/http"
	"sync"

	"github.com/gernest/tt/api"
	accesslog "github.com/gernest/tt/pkg/access_log"
	"github.com/gernest/tt/pkg/hrf"
	"github.com/gernest/tt/pkg/proxy"
	"github.com/gernest/tt/zlg"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var _ proxy.Proxy = (*Proxy)(nil)

type ListenContext struct {
	Cancel      context.CancelFunc
	HandlerChan chan http.Handler
	Listener    net.Listener
}

type Proxy struct {
	opts         *proxy.Options
	config       *api.Config
	mu           sync.Mutex
	context      map[string]*ListenContext
	ctx          context.Context
	accessLogger *accesslog.Access
	zlg          *zap.Logger
}

func (p *Proxy) configure(x *api.Config) error {
	// avoid wasteful reloads by making sure that the configuration changed
	if !proto.Equal(p.config, x) {
		return p.reload(x)
	}
	return nil
}

func (p *Proxy) reload(a *api.Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config = a
	return p.build(p.ctx)
}

func (p *Proxy) Get(ctx context.Context) (*api.Config, error) {
	return p.config, nil
}

func (p *Proxy) Put(ctx context.Context, config *api.Config) error {
	return p.configure(config)
}

func clone(a *api.Config) *api.Config {
	return proto.Clone(a).(*api.Config)
}

func (p *Proxy) Post(ctx context.Context, config *api.Config) error {
	old := clone(p.config)
	m := make(map[string]*api.Route)
	for i := 0; i < len(old.Routes); i++ {
		m[old.Routes[i].Name] = old.Routes[i]
	}
	for _, n := range config.Routes {
		if r, ok := m[n.Name]; ok {
			// Update existing route by replacing the old one with the new route.
			*r = *n
		} else {
			old.Routes = append(old.Routes, n)
		}
	}
	return p.configure(old)
}

func (p *Proxy) Delete(ctx context.Context, config *api.Config) error {
	old := clone(p.config)
	m := make(map[string]*api.Route)
	for i := 0; i < len(old.Routes); i++ {
		m[old.Routes[i].Name] = old.Routes[i]
	}
	for _, n := range config.Routes {
		if r, ok := m[n.Name]; ok {
			delete(m, r.Name)
		}
	}
	x := &api.Config{}
	for _, n := range m {
		x.Routes = append(x.Routes, n)
	}
	return p.configure(x)
}

func (p *Proxy) Config() proxy.Config {
	return p
}

func (p *Proxy) Boot(ctx context.Context, config *proxy.Options) error {
	p.zlg = zlg.Logger.Named("http")
	p.opts = config
	p.context = make(map[string]*ListenContext)
	p.config = &config.Routes
	return p.build(ctx)
}

func (p *Proxy) build(ctx context.Context) error {
	p.zlg.Info("Building routes")
	// group routes by host:port
	m := map[string][]*api.Route{}
	for _, r := range p.config.Routes {
		// only pick http services
		if r.Protocol != api.Protocol_HTTP {
			continue
		}
		h := proxy.BindToHostPort(r.Bind, p.opts.Listen.HTTP.HostPort)
		m[h] = append(m[h], r)
	}
	h := make(map[string]*mux.Router)
	var newListeners []string
	for k, v := range m {
		x, err := p.Handler(v)
		if err != nil {
			return err
		}
		h[k] = x
		newListeners = append(newListeners, k)
	}

	p.zlg.Info("Starting listenrets")
	if err := p.listen(newListeners...); err != nil {
		return err
	}
	if p.accessLogger == nil {
		// we only need one instance of this. No need to create a new one upon
		// reloading of routes
		p.zlg.Info("Starting access log")
		p.accessLogger = accesslog.New(p.opts.AccessLog,
			&accesslog.Zap{
				Logger: p.zlg.Named("access"),
			},
		)
		go p.accessLogger.Run(ctx)
	}

	// register handlers
	for k, v := range h {
		ls := p.context[k]
		if ls.Cancel != nil {
			// this was already registered before just update the handler
			ls.HandlerChan <- v
			continue
		}
		ctx, cancel := context.WithCancel(ctx)
		ls.Cancel = cancel
		p.bindServer(ctx, ls, v)
	}
	return nil
}

func (p *Proxy) bindServer(ctx context.Context, ln *ListenContext, base *mux.Router) {
	p.zlg.Info("Starting  server", zap.String("addr", ln.Listener.Addr().String()))
	base.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		path, _ := route.GetPathTemplate()
		p.zlg.Info(path)
		return nil
	})
	// set logger
	ctx = zlg.Set(ctx, p.zlg)
	svr := &http.Server{
		BaseContext: func(l net.Listener) context.Context { return ctx },
		Handler:     NewDynamic(ctx, ln.HandlerChan, base),
	}
	go func() {
		svr.Serve(ln.Listener)
	}()
}

func (p *Proxy) listen(ls ...string) error {
	m := make(map[string]*ListenContext)
	for _, l := range ls {
		if _, ok := p.context[l]; ok {
			// if we already had a listener we don't want to bind multiple listeners on
			// the sape port
			continue
		}
		ln, err := net.Listen("tcp", l)
		if err != nil {
			return err
		}
		m[l] = &ListenContext{
			HandlerChan: make(chan http.Handler),
			Listener:    ln,
		}
	}
	// update context to reflect the new changes
	for k, v := range m {
		p.context[k] = v
	}
	for k, v := range p.context {
		if _, ok := m[k]; !ok {
			// This listener was removed we need to properly cear everything associated
			// with this listener.
			if v.Cancel != nil {
				v.Cancel()
			}
			if err := v.Listener.Close(); err != nil {
				return err
			}
			delete(p.context, k)
		}
	}
	// By now we have context reflecting the new desired state.
	return nil
}

func (p *Proxy) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for k, v := range p.context {
		if v.Cancel != nil {
			v.Cancel()
		}
		v.Listener.Close()
		delete(p.context, k)
	}
	return nil
}

func (p *Proxy) Health() hrf.Health {
	p.mu.Lock()
	defer p.mu.Unlock()
	return hrf.Health{
		Status:    hrf.Pass,
		Version:   p.opts.Info.Version,
		ReleaseID: p.opts.Info.ReleaseID,
		ServiceID: p.opts.Info.ID,
	}
}
