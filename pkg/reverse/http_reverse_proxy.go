package reverse

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/gernest/tt/api"
	"github.com/gernest/tt/pkg/balance"
	"github.com/gernest/tt/pkg/meta"
)

type Director interface {
	// Director is a function which modifies
	// the request into a new request to be sent
	// using Transport. Its response is then copied
	// back to the original client unmodified.
	// Director must not access the provided Request
	// after returning.
	Request(r *http.Request)
}

type DirectorFunc func(r *http.Request)

func (df DirectorFunc) Request(r *http.Request) {
	df(r)
}

// Request modifies req to point to target
func Request(target *url.URL, req *http.Request) {
	targetQuery := target.RawQuery
	req.URL.Scheme = target.Scheme
	req.URL.Host = target.Host
	req.URL.Path, req.URL.RawPath = joinURLPath(target, req.URL)
	if targetQuery == "" || req.URL.RawQuery == "" {
		req.URL.RawQuery = targetQuery + req.URL.RawQuery
	} else {
		req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
	}
	if _, ok := req.Header["User-Agent"]; !ok {
		// explicitly disable User-Agent so it's not set to default value
		req.Header.Set("User-Agent", "")
	}
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

func joinURLPath(a, b *url.URL) (path, rawpath string) {
	if a.RawPath == "" && b.RawPath == "" {
		return singleJoiningSlash(a.Path, b.Path), ""
	}
	// Same as singleJoiningSlash, but uses EscapedPath to determine
	// whether a slash should be added
	apath := a.EscapedPath()
	bpath := b.EscapedPath()

	aslash := strings.HasSuffix(apath, "/")
	bslash := strings.HasPrefix(bpath, "/")

	switch {
	case aslash && bslash:
		return a.Path + b.Path[1:], apath + bpath[1:]
	case !aslash && !bslash:
		return a.Path + "/" + b.Path, apath + "/" + bpath
	}
	return a.Path + b.Path, apath + bpath
}

func DirectorFromLoadBalance(ba balance.Balance) Director {
	return DirectorFunc(func(r *http.Request) {
		target := ba.Next()
		if m := meta.GetMetics(r.Context()); m != nil {
			m.Target = target.URL.String()
		}
		Request(target.URL, r)
	})
}

func New(route *api.Route) (*httputil.ReverseProxy, error) {
	ba, err := balance.FromRoute(route)
	if err != nil {
		return nil, err
	}
	direct := DirectorFromLoadBalance(ba)
	return &httputil.ReverseProxy{Director: direct.Request}, nil
}
