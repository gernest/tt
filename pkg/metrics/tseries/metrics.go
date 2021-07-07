package tseries

import (
	"context"
	"net/http"

	"github.com/gernest/tt/pkg/xhttp/xlabels"
	"github.com/prometheus/client_golang/prometheus"
)

type metricsKey struct{}

type Metrics struct {
	Route, Service string
	Request        Request
}

type Request struct {
	UserAgent UserAgent
}

type UserAgent struct {
	Name      string
	Version   string
	OS        string
	OSVersion string
	Device    string
	Mobile    bool
	Tablet    bool
	Desktop   bool
	Bot       bool
	URL       string
	String    string
}

func SetMetric(ctx context.Context, m *Metrics) context.Context {
	return context.WithValue(ctx, metricsKey{}, m)
}

func GetMetics(ctx context.Context) *Metrics {
	return ctx.Value(metricsKey{}).(*Metrics)
}

func (m *Metrics) Labels(
	r *http.Request,
	code int,
) prometheus.Labels {
	return prometheus.Labels{
		xlabels.Code:   sanitizeCode(code),
		xlabels.Method: sanitizeMethod(r.Method),
		xlabels.Path:   r.URL.Path,
	}
}
