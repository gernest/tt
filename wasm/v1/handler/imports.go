package handler

import (
	proxywasm "mosn.io/proxy-wasm-go-host/proxywasm/v1"
)

var _ proxywasm.ImportsHandler = (*Wasm)(nil)

type Wasm struct {
	Zap
	Base
	Request
	Response
	HTTPCall
	L4
	Plugin
	FFI
	GRPC
	Metrics
	Queue
	KeyValue
	CustomExtension
	SharedData
}

func (d *Wasm) Clone() *Wasm {
	return &Wasm{}
}
