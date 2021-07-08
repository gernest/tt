package imports

import (
	"mosn.io/proxy-wasm-go-host/proxywasm/common"
	x "mosn.io/proxy-wasm-go-host/proxywasm/v2"
)

var _ x.ImportsHandler = Imports(nil)

type Imports interface {
	Base
	Plugin
	Buffer
	L4
	HTTP
	KeyValue
	Queue
	Timer
	Metrics
	GRPC
	FFI
}
type Base interface {
	// for golang host environment
	// Wait until async call return, eg. sync http call in golang
	Wait() x.Action
	// integration
	Log(logLevel x.LogLevel, msg string) x.Result
	SetEffectiveContext(contextID int32) x.Result
	ContextFinalize() x.Result
}

type Plugin interface {
	GetPluginConfig() common.IoBuffer
	GetVmConfig() common.IoBuffer
}

type Buffer interface {
	GetCustomBuffer(bufferType x.BufferType) common.IoBuffer
	GetCustomMap(mapType x.MapType) common.HeaderMap
}

type L4 interface {
	GetDownStreamData() common.IoBuffer
	GetUpstreamData() common.IoBuffer
	ResumeDownStream() x.Result
	ResumeUpStream() x.Result

	CloseDownStream() x.Result
	CloseUpStream() x.Result

	ResumeCustomStream(streamType x.StreamType) x.Result
	CloseCustomStream(streamType x.StreamType) x.Result
}

type HTTPRequest interface {
	GetHttpRequestHeader() common.HeaderMap
	GetHttpRequestBody() common.IoBuffer
	GetHttpRequestTrailer() common.HeaderMap
	GetHttpRequestMetadata() common.HeaderMap
}

type HTTPResponse interface {
	GetHttpResponseHeader() common.HeaderMap
	GetHttpResponseBody() common.IoBuffer
	GetHttpResponseTrailer() common.HeaderMap
	GetHttpResponseMetadata() common.HeaderMap
}

type HTTP interface {
	HTTPRequest
	HTTPResponse

	GetHttpCallResponseHeaders() common.HeaderMap
	GetHttpCalloutResponseBody() common.IoBuffer
	GetHttpCallResponseTrailer() common.HeaderMap
	GetHttpCallResponseMetadata() common.HeaderMap

	SendHttpResp(responseCode int32, responseCodeDetails common.IoBuffer, responseBody common.IoBuffer,
		additionalHeadersMap common.HeaderMap, grpcStatus int32) x.Result
	DispatchHttpCall(upstream string, headersMap common.HeaderMap, bodyData common.IoBuffer,
		trailersMap common.HeaderMap, timeoutMilliseconds uint32) (uint32, x.Result)

	ResumeHttpRequest() x.Result
	ResumeHttpResponse() x.Result

	CloseHttpRequest() x.Result
	CloseHttpResponse() x.Result
}

type KeyValue interface {
	OpenSharedKvstore(kvstoreName string, createIfNotExist bool) (uint32, x.Result)
	GetSharedKvstore(kvstoreID uint32) x.KVStore
	DeleteSharedKvstore(kvstoreID uint32) x.Result
}

type Queue interface {
	OpenSharedQueue(queueName string, createIfNotExist bool) (uint32, x.Result)
	DequeueSharedQueueItem(queueID uint32) (string, x.Result)
	EnqueueSharedQueueItem(queueID uint32, payload string) x.Result
	DeleteSharedQueue(queueID uint32) x.Result
}

type Timer interface {
	CreateTimer(period int32, oneTime bool) (uint32, x.Result)
	DeleteTimer(timerID uint32) x.Result
}

type Metrics interface {
	CreateMetric(metricType x.MetricType, metricName string) (uint32, x.Result)
	GetMetricValue(metricID uint32) (int64, x.Result)
	SetMetricValue(metricID uint32, value int64) x.Result
	IncrementMetricValue(metricID uint32, offset int64) x.Result
	DeleteMetric(metricID uint32) x.Result
}

type GRPC interface {
	DispatchGrpcCall(upstream string, serviceName string, serviceMethod string, initialMetadataMap common.HeaderMap,
		grpcMessage common.IoBuffer, timeoutMilliseconds uint32) (uint32, x.Result)
	OpenGrpcStream(upstream string, serviceName string, serviceMethod string, initialMetadataMap common.HeaderMap) (uint32, x.Result)
	SendGrpcStreamMessage(calloutID uint32, grpcMessageData common.IoBuffer) x.Result
	CancelGrpcCall(calloutID uint32) x.Result
	CloseGrpcCall(calloutID uint32) x.Result
}

type FFI interface {
	CallCustomFunction(customFunctionID uint32, parametersData string) (string, x.Result)
}
