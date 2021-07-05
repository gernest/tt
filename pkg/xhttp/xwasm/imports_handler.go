package xwasm

import (
	"net/http"

	"mosn.io/proxy-wasm-go-host/proxywasm/common"
)

type Imports struct {
	Request  *http.Request
	Response http.ResponseWriter
}

func (i *Imports) GetHttpRequestHeader() common.HeaderMap {
	return &Header{head: i.Request.Header}
}
