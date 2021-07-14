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
	Config    *api.Middleware_Wasm_Config
	NewBuffer func() *buffers.IO
}

func safeBuffer() (create func() *buffers.IO, release func()) {
	var ls []*buffers.IO
	return func() *buffers.IO {
			b := buffers.Get()
			ls = append(ls, b)
			return b
		}, func() {
			buffers.Put(ls...)
		}
}

func (p *Plugin) GetPluginConfig() common.IoBuffer {
	if p.Config == nil {
		return nil
	}
	if p.NewBuffer == nil {
		return nil
	}
	if x := p.Config.GetPlugin(); x != nil {
		buf := p.NewBuffer()
		m.Marshal(buf, x)
		return buf
	}
	return nil
}

func (p *Plugin) GetVmConfig() common.IoBuffer {
	if p.Config == nil {
		return nil
	}
	if p.NewBuffer == nil {
		return nil
	}
	if i := p.Config.GetInstance(); i != nil {
		buf := p.NewBuffer()
		m.Marshal(buf, i)
		return buf
	}
	return nil
}
