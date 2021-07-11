package tseries

import (
	"github.com/gernest/tt/pkg/zlg"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/zap"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prometheus/tsdb"
)

func open(c *Config) (*tsdb.DB, error) {
	opts := c.Options()
	stats := tsdb.NewDBStats()
	return tsdb.Open(
		c.Path,
		logger(),
		prometheus.DefaultRegisterer,
		opts,
		stats,
	)
}

func logger() log.Logger {
	return zap.NewZapSugarLogger(zlg.Logger.Named("tsdb"), zlg.Level.Level())
}

type Store struct {
	db *tsdb.DB
}

func (db *Store) Appender(metrics []*dto.MetricFamily) {
}
