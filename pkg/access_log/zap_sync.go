package accesslog

import "go.uber.org/zap"

var _ Sync = (*Zap)(nil)

type Zap struct {
	Logger *zap.Logger
}

func (Zap) sync() {}

func (z *Zap) Sync(e *Entry) {
	defer e.Release()
	z.Logger.Debug(e.Request.Path, e.fields()...)
}

func (e *Entry) fields() (ls []zap.Field) {
	ls = append(ls,
		zap.Int32("status", e.Response.StatusCode),
		zap.String("route", e.Info.Route),
		zap.String("service", e.Info.Service),
		zap.Strings("host", e.Info.VirtualHosts),
	)
	return
}
