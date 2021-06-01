package proxy

import (
	"context"
	"net"

	"github.com/gernest/tt/api"
	"go.uber.org/zap"
)

type middleareFunc func(Target) Target

type chain []middleareFunc

func (c chain) then(t Target) Target {
	for _, h := range c {
		t = h(t)
	}
	return t
}

func buildMiddleares(r *api.Route) chain {
	c := chain{}
	if len(r.MetricsLabels) > 0 {
		// Inject metrics labels to targets context meta
		c = append(c, func(t Target) Target {
			m := &metricsLabelsTarget{
				target: t,
				labels: make(map[string]string),
				logger: zap.L().Named("metrics.label"),
			}
			for k, v := range r.MetricsLabels {
				m.labels[k] = v
			}
			return m
		})
	}
	return c
}

// metricsLabelsTarget injects upstream labels
type metricsLabelsTarget struct {
	target Target
	labels map[string]string
	logger *zap.Logger
}

func (m *metricsLabelsTarget) HandleConn(ctx context.Context, conn net.Conn) {
	m.logger.Info("Adding labels")
	ctx = UpdateContext(ctx, func(cm *ContextMeta) {
		cm.Labels = m.labels
	})
	m.target.HandleConn(ctx, conn)
}
