package xwasm

import (
	"github.com/gernest/tt/wasm/imports"
	"go.uber.org/zap"
	x "mosn.io/proxy-wasm-go-host/proxywasm/v2"
)

var _ imports.Logger = nil

type Zap struct {
	L *zap.Logger
}

func (z *Zap) Log(logLevel x.LogLevel, msg string) x.Result {
	if z.L == nil {
		return nyet()
	}
	switch logLevel {
	case x.LogLevelTrace:
	case x.LogLevelDebug:
		z.L.Debug(msg)
	case x.LogLevelInfo:
		z.L.Info(msg)
	case x.LogLevelWarning:
		z.L.Warn(msg)
	case x.LogLevelError:
		z.L.Error(msg)
	}
	return x.ResultOk
}
