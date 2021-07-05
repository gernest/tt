package exports

import (
	x "mosn.io/proxy-wasm-go-host/proxywasm/v2"
)

var _ x.Exports = Exports(nil)

type Exports interface {
	Integration
	Context
	Configuration
	L4
	HTTP
	GRPC
	Timer
	Queue
	FFI
}

type Integration interface {
	ProxyOnMemoryAllocate(memorySize int32) (int32, error)
}

type Context interface {
	ProxyOnContextCreate(contextID int32, parentContextID int32, contextType x.ContextType) (int32, error)
	ProxyOnContextFinalize(contextID int32) (int32, error)
}

type Configuration interface {
	ProxyOnVmStart(vmID int32, vmConfigurationSize int32) (int32, error)
	ProxyOnPluginStart(pluginID int32, pluginConfigurationSize int32) (int32, error)
}

type L4 interface {
	ProxyOnNewConnection(streamID int32) (x.Action, error)
	ProxyOnDownstreamData(streamID int32, dataSize int32, endOfStream int32) (x.Action, error)
	ProxyOnDownstreamClose(contextID int32, closeSource x.CloseSourceType) error
	ProxyOnUpstreamData(streamID int32, dataSize int32, endOfStream int32) (x.Action, error)
	ProxyOnUpstreamClose(streamID int32, closeSource x.CloseSourceType) error
}

type HTTP interface {
	ProxyOnHttpRequestHeaders(streamID int32, numHeaders int32, endOfStream int32) (x.Action, error)
	ProxyOnHttpRequestBody(streamID int32, bodySize int32, endOfStream int32) (x.Action, error)
	ProxyOnHttpRequestTrailers(streamID int32, numTrailers int32, endOfStream int32) (x.Action, error)
	ProxyOnHttpRequestMetadata(streamID int32, numElements int32) (x.Action, error)

	ProxyOnHttpResponseHeaders(streamID int32, numHeaders int32, endOfStream int32) (x.Action, error)
	ProxyOnHttpResponseBody(streamID int32, bodySize int32, endOfStream int32) (x.Action, error)
	ProxyOnHttpResponseTrailers(streamID int32, numTrailers int32, endOfStream int32) (x.Action, error)
	ProxyOnHttpResponseMetadata(streamID int32, numElements int32) (x.Action, error)

	ProxyOnHttpCallResponse(calloutID int32, numHeaders int32, bodySize int32, numTrailers int32) error
}

type Queue interface {
	ProxyOnQueueReady(queueID int32) error
}

type Timer interface {
	ProxyOnTimerReady(timerID int32) error
}

type GRPC interface {
	ProxyOnGrpcCallResponseHeaderMetadata(calloutID int32, numHeaders int32) error
	ProxyOnGrpcCallResponseMessage(calloutID int32, messageSize int32) error
	ProxyOnGrpcCallResponseTrailerMetadata(calloutID int32, numTrailers int32) error
	ProxyOnGrpcCallClose(calloutID int32, statusCode int32) error
}

type FFI interface {
	ProxyOnCustomCallback(customCallbackID int32, parametersSize int32) (int32, error)
}
