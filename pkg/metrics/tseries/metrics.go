package tseries

import (
	"context"
	"net/http"
	"time"

	"github.com/prometheus/prometheus/pkg/labels"
)

type metricsKey struct{}

type Metrics struct {
	Route, Service string
	Request        Request
	Response       Response
}

type Request struct {
	Path      string
	UserAgent UserAgent
	Size      int64
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

type Response struct {
	Size     int64
	Code     int
	Duration time.Duration
}

func SetMetric(ctx context.Context, m *Metrics) context.Context {
	return context.WithValue(ctx, metricsKey{}, m)
}

func GetMetics(ctx context.Context) *Metrics {
	return ctx.Value(metricsKey{}).(*Metrics)
}

func (m *Metrics) RecordRequest(r *http.Request) {}

func (m *Metrics) Labels() labels.Labels {
	return labels.New()
}
