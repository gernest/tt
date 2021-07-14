package exports

import (
	x "mosn.io/proxy-wasm-go-host/proxywasm/v1"
)

var _ x.Exports = Exports(nil)

type Exports interface {
	Integration
	Context
	Configuration
	L4
	HTTPRequest
	HTTPResponse
	GRPC
	Timer
	Queue
}

type Integration interface {
	ProxyOnDone(contextID int32) (int32, error)
	ProxyOnLog(contextID int32) error
	ProxyOnDelete(contextID int32) error
	ProxyOnMemoryAllocate(memorySize int32) (int32, error)
}

type Context interface {
	ProxyOnContextCreate(contextID int32, parentContextID int32) error
	ProxyOnContextFinalize(contextID int32) (int32, error)
}

type Configuration interface {
	ProxyOnVmStart(rootContextID int32, vmConfigurationSize int32) (int32, error)
	ProxyOnConfigure(rootContextID int32, pluginConfigurationSize int32) (int32, error)
}

type L4 interface {
	ProxyOnNewConnection(contextID int32) (x.Action, error)
	ProxyOnDownstreamData(contextID int32, dataLength int32, endOfStream int32) (x.Action, error)
	ProxyOnDownstreamConnectionClose(contextID int32, closeType int32) error
	ProxyOnUpstreamData(contextID int32, dataLength int32, endOfStream int32) (x.Action, error)
	ProxyOnUpstreamConnectionClose(contextID int32, closeType int32) error
}

type HTTPRequest interface {
	ProxyOnRequestHeaders(contextID int32, headers int32, endOfStream int32) (x.Action, error)
	ProxyOnRequestBody(contextID int32, bodyBufferLength int32, endOfStream int32) (x.Action, error)
	ProxyOnRequestTrailers(contextID int32, trailers int32) (x.Action, error)
	ProxyOnRequestMetadata(contextID int32, nElements int32) (x.Action, error)
}
type HTTPResponse interface {
	ProxyOnResponseHeaders(contextID int32, headers int32, endOfStream int32) (x.Action, error)
	ProxyOnResponseBody(contextID int32, bodyBufferLength int32, endOfStream int32) (x.Action, error)
	ProxyOnResponseTrailers(contextID int32, trailers int32) (x.Action, error)
	ProxyOnResponseMetadata(contextID int32, nElements int32) (x.Action, error)

	ProxyOnHttpCallResponse(contextID int32, token int32, headers int32, bodySize int32, trailers int32) error
}

type Queue interface {
	ProxyOnQueueReady(rootContextID int32, token int32) error
}

type Timer interface {
	ProxyOnTick(rootContextID int32) error
}

type GRPC interface {
	ProxyOnGrpcCallResponseHeaderMetadata(contextID int32, calloutID int32, nElements int32) error
	ProxyOnGrpcCallResponseMessage(contextID int32, calloutID int32, msgSize int32) error
	ProxyOnGrpcCallResponseTrailerMetadata(contextID int32, calloutID int32, nElements int32) error
	ProxyOnGrpcCallClose(contextID int32, calloutID int32, statusCode int32) error
}
