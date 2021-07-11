package accesslog

import (
	"context"
	"sync"

	"github.com/gernest/tt/api"
)

var pool = &sync.Pool{
	New: func() interface{} {
		return new(Entry)
	},
}

type Entry struct {
	api.AccessEntry
}

func (e *Entry) Release() {
	e.reset()
	pool.Put(e)
}

func (e *Entry) reset() {
	e.Route = ""
	e.Service = ""
	e.VirtualHost = ""
	e.resetRequest()
	e.resetResponse()
	e.resetReverseProxy()
}

func (e *Entry) resetRequest() {
	if r := e.GetRequest(); r != nil {
		if ua := r.GetUserAgent(); ua != nil {
			ua.Name = ""
			ua.Version = ""
			ua.Os = ""
			ua.OsVersion = ""
			ua.Device = ""
			ua.Tablet = false
			ua.Mobile = false
			ua.Desktop = false
		}
		r.Method = ""
		r.Path = ""
		r.Size = 0
	}
}

func (e *Entry) resetResponse() {
	if r := e.GetResponse(); r != nil {
		r.Size = 0
		r.StatusCode = 0
	}
}

func (e *Entry) resetReverseProxy() {
	if r := e.GetReverseProxy(); r != nil {
		r.BytesSent = 0
		r.BytesReceived = 0
	}
}

type Access struct {
	sync    Sync
	in, out chan *Entry
}

func (a *Access) Start(ctx context.Context) {
}

func New(opts RingOptions, syncer Sync) *Access {
	return &Access{
		in:  make(chan *Entry, opts.InSize),
		out: make(chan *Entry, opts.OutSize),
	}
}

func (a *Access) Run(ctx context.Context) {
	ring := &Ring{In: a.in, Out: a.out}
	go ring.Run(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-a.out:
			if !ok {
				return
			}
			a.sync.Sync(e)
		}
	}
}

func (a *Access) Record(e *Entry) {
	a.in <- e
}
