package xwasm

import (
	"github.com/gernest/tt/wasm/imports"
	x "mosn.io/proxy-wasm-go-host/proxywasm/v2"
)

var _ imports.Base = (*Base)(nil)

type Base struct{}

func (b *Base) SetEffectiveContext(contextID int32) x.Result {
	return x.ResultOk
}

func (b *Base) ContextFinalize() x.Result {
	return x.ResultOk
}

func (b *Base) Wait() x.Action {
	return x.ActionContinue
}
