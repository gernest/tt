package xwasm

import (
	"net/http"

	"github.com/gernest/tt/wasm/imports"
	"mosn.io/proxy-wasm-go-host/proxywasm/common"
)

var _ imports.HTTPRequest = (*Request)(nil)

type Request struct {
	Request *http.Request
}

func (r *Request) GetHttpRequestHeader() common.HeaderMap {
	return &Header{head: r.Request.Header}
}

func (r *Request) GetHttpRequestBody() common.IoBuffer {
	if io, ok := r.Request.Body.(common.IoBuffer); ok {
		return io
	}
	return nil
}

func (r *Request) GetHttpRequestTrailer() common.HeaderMap {
	return &Header{head: r.Request.Trailer}
}

func (r *Request) GetHttpRequestMetadata() common.HeaderMap {
	return &Header{head: r.Request.Trailer}
}
