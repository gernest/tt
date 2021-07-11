package accesslog

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gernest/tt/api"
	"github.com/gernest/tt/zlg"
	"github.com/golang/protobuf/ptypes"
	ua "github.com/mileusna/useragent"
	"go.uber.org/atomic"
)

var pool = &sync.Pool{
	New: func() interface{} {
		return &Entry{
			AccessEntry: api.AccessEntry{
				Request: &api.AccessEntry_Request{
					UserAgent: &api.AccessEntry_UserAgent{},
				},
				Response:     &api.AccessEntry_Response{},
				ReverseProxy: &api.AccessEntry_ReverseProxy{},
				Info:         &api.AccessEntry_Info{},
			},
		}
	},
}

type Entry struct {
	api.AccessEntry
}

func NewEntry() *Entry {
	return pool.Get().(*Entry)
}

func (e *Entry) Release() {
	e.reset()
	pool.Put(e)
}

func (e *Entry) reset() {
	e.resetInfo()
	e.resetRequest()
	e.resetResponse()
	e.resetReverseProxy()
	e.Duration = nil
}
func (e *Entry) resetInfo() {
	e.Info.Route = ""
	e.Info.Service = ""
	e.Info.VirtualHost = ""
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
		r.TimeToWriteHeader = nil
	}
}

func (e *Entry) resetReverseProxy() {
	if r := e.GetReverseProxy(); r != nil {
		r.BytesSent = 0
		r.BytesReceived = 0
		r.Target = ""
	}
}

func (e *Entry) Update(
	r *http.Request,
	statusCode int,
	responseSize int64,
	duration time.Duration,
	durationToWriteHeader time.Duration,
) {
	// user agent
	agent := ua.Parse(r.UserAgent())
	e.Request.UserAgent.Name = agent.Name
	e.Request.UserAgent.Version = agent.Version
	e.Request.UserAgent.Os = agent.OS
	e.Request.UserAgent.OsVersion = agent.OSVersion
	e.Request.UserAgent.Device = agent.Device
	e.Request.UserAgent.Mobile = agent.Mobile
	e.Request.UserAgent.Tablet = agent.Tablet
	e.Request.UserAgent.Desktop = agent.Desktop
	e.Request.UserAgent.Desktop = agent.Desktop
	e.Request.UserAgent.Bot = agent.Bot

	//request
	e.Request.Method = sanitizeMethod(r.Method)
	e.Request.Path = sanitizeMethod(r.URL.Path)

	//response
	e.Response.StatusCode = int32(statusCode)
	e.Response.Size = responseSize
	e.Response.TimeToWriteHeader = ptypes.DurationProto(durationToWriteHeader)

	// misc
	e.Duration = ptypes.DurationProto(duration)
}

func sanitizeMethod(m string) string {
	switch m {
	case "GET", "get":
		return "get"
	case "PUT", "put":
		return "put"
	case "HEAD", "head":
		return "head"
	case "POST", "post":
		return "post"
	case "DELETE", "delete":
		return "delete"
	case "CONNECT", "connect":
		return "connect"
	case "OPTIONS", "options":
		return "options"
	case "NOTIFY", "notify":
		return "notify"
	default:
		return strings.ToLower(m)
	}
}

// If the wrapped http.Handler has not set a status code, i.e. the value is
// currently 0, santizeCode will return 200, for consistency with behavior in
// the stdlib.
func sanitizeCode(s int) string {
	switch s {
	case 100:
		return "100"
	case 101:
		return "101"

	case 200, 0:
		return "200"
	case 201:
		return "201"
	case 202:
		return "202"
	case 203:
		return "203"
	case 204:
		return "204"
	case 205:
		return "205"
	case 206:
		return "206"

	case 300:
		return "300"
	case 301:
		return "301"
	case 302:
		return "302"
	case 304:
		return "304"
	case 305:
		return "305"
	case 307:
		return "307"

	case 400:
		return "400"
	case 401:
		return "401"
	case 402:
		return "402"
	case 403:
		return "403"
	case 404:
		return "404"
	case 405:
		return "405"
	case 406:
		return "406"
	case 407:
		return "407"
	case 408:
		return "408"
	case 409:
		return "409"
	case 410:
		return "410"
	case 411:
		return "411"
	case 412:
		return "412"
	case 413:
		return "413"
	case 414:
		return "414"
	case 415:
		return "415"
	case 416:
		return "416"
	case 417:
		return "417"
	case 418:
		return "418"

	case 500:
		return "500"
	case 501:
		return "501"
	case 502:
		return "502"
	case 503:
		return "503"
	case 504:
		return "504"
	case 505:
		return "505"

	case 428:
		return "428"
	case 429:
		return "429"
	case 431:
		return "431"
	case 511:
		return "511"

	default:
		return strconv.Itoa(s)
	}
}

type Access struct {
	sync    Sync
	in, out chan *Entry
	stopped atomic.Bool
}

func New(opts Options, logSync Sync) *Access {
	return &Access{
		in:   make(chan *Entry, opts.InSize),
		out:  make(chan *Entry, opts.OutSize),
		sync: logSync,
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

func (a *Access) Close() error {
	a.stopped.Store(true)
	close(a.in)
	close(a.out)
	return nil
}

func (a *Access) Record(e *Entry) {
	zlg.Info("Recording entry")
	a.in <- e
}

type accessLogKey struct{}

var oblivion = BlackHole{}

func Get(ctx context.Context) Recorder {
	if a := ctx.Value(accessLogKey{}); a != nil {
		return a.(*Access)
	}
	return oblivion
}

func Set(ctx context.Context, a *Access) context.Context {
	return context.WithValue(ctx, accessLogKey{}, a)
}

type Recorder interface {
	Record(e *Entry)
}
