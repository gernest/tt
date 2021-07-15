package handler

import (
	"github.com/gernest/tt/wasm/v1/imports"
	"go.uber.org/zap"
	x "mosn.io/proxy-wasm-go-host/proxywasm/v1"
)

var _ imports.Logger = nil

type Zap struct {
	L *zap.Logger
}

func (z *Zap) Log(logLevel x.LogLevel, msg string) x.WasmResult {
	if z.L == nil {
		return x.WasmResultUnimplemented
	}
	switch logLevel {
	case x.LogLevelTrace:
	case x.LogLevelDebug:
		z.L.Debug(msg)
	case x.LogLevelInfo:
		z.L.Info(msg)
	case x.LogLevelWarn:
		z.L.Warn(msg)
	case x.LogLevelError:
		z.L.Error(msg)
	}
	return x.WasmResultOk
}
