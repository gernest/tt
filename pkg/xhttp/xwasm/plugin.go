package xwasm

import (
	"github.com/gernest/tt/api"
	"github.com/gernest/tt/pkg/buffers"
	"github.com/gernest/tt/wasm/imports"
	"github.com/golang/protobuf/jsonpb"
	"mosn.io/proxy-wasm-go-host/proxywasm/common"
)

var _ imports.Plugin = (*Plugin)(nil)
var m jsonpb.Marshaler

type Plugin struct {
	MW        *api.Middleware_Wasm
	NewBuffer func() *buffers.IO
}

func (p *Plugin) GetPluginConfig() common.IoBuffer {
	buf := p.NewBuffer()
	m.Marshal(buf, p.MW.GetConfig().Plugin)
	return buf
}

func (p *Plugin) GetVmConfig() common.IoBuffer {
	buf := p.NewBuffer()
	m.Marshal(buf, p.MW.GetConfig().Instance)
	return buf
}
