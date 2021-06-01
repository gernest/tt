package proxy

import (
	"time"

	"github.com/gernest/tt/zlg"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var totalTCPRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "tcp_requests_total",
		Help: "Total number of tcp requests",
	},
	[]string{},
)

func (m *ContextMeta) Complete() {
	m.End = time.Now()
	m.Log()
	m.Emit()
}

// Emit based on m emits various metrics exposed by prometheus
func (m *ContextMeta) Emit() {
	base := m.GetBaseLabels()
	totalTCPRequests.With(base).Inc()
}

// Log write logs about the request
func (m *ContextMeta) Log() {
	fields := []zapcore.Field{
		zap.String("server_name", m.ServerName.String()),
		zap.Bool("acme", m.ACME.Load()),
		zap.Bool("fixed", m.ACME.Load()),
		zap.String("protocol", Protocol(m.Protocol.Load()).String()),
		zap.Duration("duration", m.End.Sub(m.Start)),
	}
	if m.NoMatch.Load() {
		zlg.Info("PASS", fields...)
		return
	}
	zlg.Info("FAILED", fields...)
}

func (m *ContextMeta) GetBaseLabels(lbs ...map[string]string) prometheus.Labels {
	lbs = append(lbs, map[string]string{
		"acme":     m.ACME.String(),
		"fixed":    m.Fixed.String(),
		"no_match": m.NoMatch.String(),
		"protocol": Protocol(m.Protocol.Load()).String(),
		"route":    m.RouteName.String(),
	})
	return m.GetLabels(lbs...)
}
func (m *ContextMeta) GetLabels(lbs ...map[string]string) prometheus.Labels {
	x := make(map[string]string)
	for k, v := range m.Labels {
		x[k] = v
	}
	for _, lb := range lbs {
		for k, v := range lb {
			x[k] = v
		}
	}

	return x
}
