package xwasm

import (
	"mosn.io/proxy-wasm-go-host/proxywasm/common"
	proxywasm "mosn.io/proxy-wasm-go-host/proxywasm/v2"
)

var _ proxywasm.ImportsHandler = (*Wasm)(nil)

type Wasm struct {
	Zap
	Request
}

func (d *Wasm) Clone() *Wasm {
	return &Wasm{}
}

func (d *Wasm) Wait() proxywasm.Action { return proxywasm.ActionContinue }

func nyet() proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) SetEffectiveContext(contextID int32) proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) ContextFinalize() proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) ResumeDownStream() proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) ResumeUpStream() proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) ResumeHttpRequest() proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) ResumeHttpResponse() proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) ResumeCustomStream(streamType proxywasm.StreamType) proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) CloseDownStream() proxywasm.Result {
	return proxywasm.ResultUnimplemented
}
func (d *Wasm) CloseUpStream() proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) CloseHttpRequest() proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) CloseHttpResponse() proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) CloseCustomStream(streamType proxywasm.StreamType) proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) SendHttpResp(responseCode int32, responseCodeDetails common.IoBuffer,
	responseBody common.IoBuffer, additionalHeadersMap common.HeaderMap, grpcStatus int32) proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) GetHttpResponseBody() common.IoBuffer { return nil }

func (d *Wasm) GetDownStreamData() common.IoBuffer { return nil }

func (d *Wasm) GetUpstreamData() common.IoBuffer { return nil }

func (d *Wasm) GetHttpCalloutResponseBody() common.IoBuffer { return nil }

func (d *Wasm) GetPluginConfig() common.IoBuffer { return nil }

func (d *Wasm) GetVmConfig() common.IoBuffer { return nil }

func (d *Wasm) GetCustomBuffer(bufferType proxywasm.BufferType) common.IoBuffer {
	return nil
}

func (d *Wasm) GetHttpResponseHeader() common.HeaderMap { return nil }

func (d *Wasm) GetHttpResponseTrailer() common.HeaderMap { return nil }

func (d *Wasm) GetHttpResponseMetadata() common.HeaderMap { return nil }

func (d *Wasm) GetHttpCallResponseHeaders() common.HeaderMap { return nil }

func (d *Wasm) GetHttpCallResponseTrailer() common.HeaderMap { return nil }

func (d *Wasm) GetHttpCallResponseMetadata() common.HeaderMap { return nil }

func (d *Wasm) GetCustomMap(mapType proxywasm.MapType) common.HeaderMap { return nil }

func (d *Wasm) OpenSharedKvstore(kvstoreName string, createIfNotExist bool) (uint32, proxywasm.Result) {
	return 0, proxywasm.ResultUnimplemented
}
func (d *Wasm) GetSharedKvstore(kvstoreID uint32) proxywasm.KVStore { return nil }

func (d *Wasm) DeleteSharedKvstore(kvstoreID uint32) proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) OpenSharedQueue(queueName string, createIfNotExist bool) (uint32, proxywasm.Result) {
	return 0, proxywasm.ResultUnimplemented
}

func (d *Wasm) DequeueSharedQueueItem(queueID uint32) (string, proxywasm.Result) {
	return "", proxywasm.ResultUnimplemented
}

func (d *Wasm) EnqueueSharedQueueItem(queueID uint32, payload string) proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) DeleteSharedQueue(queueID uint32) proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) CreateTimer(period int32, oneTime bool) (uint32, proxywasm.Result) {
	return 0, proxywasm.ResultUnimplemented
}

func (d *Wasm) DeleteTimer(timerID uint32) proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) CreateMetric(metricType proxywasm.MetricType, metricName string) (uint32, proxywasm.Result) {
	return 0, proxywasm.ResultUnimplemented
}

func (d *Wasm) GetMetricValue(metricID uint32) (int64, proxywasm.Result) {
	return 0, proxywasm.ResultUnimplemented
}

func (d *Wasm) SetMetricValue(metricID uint32, value int64) proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) IncrementMetricValue(metricID uint32, offset int64) proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) DeleteMetric(metricID uint32) proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) DispatchHttpCall(upstream string, headersMap common.HeaderMap, bodyData common.IoBuffer,
	trailersMap common.HeaderMap, timeoutMilliseconds uint32) (uint32, proxywasm.Result) {
	return 0, proxywasm.ResultUnimplemented
}

func (d *Wasm) DispatchGrpcCall(upstream string, serviceName string, serviceMethod string,
	initialMetadataMap common.HeaderMap, grpcMessage common.IoBuffer, timeoutMilliseconds uint32) (uint32, proxywasm.Result) {
	return 0, proxywasm.ResultUnimplemented
}

func (d *Wasm) OpenGrpcStream(upstream string, serviceName string, serviceMethod string,
	initialMetadataMap common.HeaderMap) (uint32, proxywasm.Result) {
	return 0, proxywasm.ResultUnimplemented
}

func (d *Wasm) SendGrpcStreamMessage(calloutID uint32, grpcMessageData common.IoBuffer) proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) CancelGrpcCall(calloutID uint32) proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) CloseGrpcCall(calloutID uint32) proxywasm.Result {
	return proxywasm.ResultUnimplemented
}

func (d *Wasm) CallCustomFunction(customFunctionID uint32, parametersData string) (string, proxywasm.Result) {
	return "", proxywasm.ResultUnimplemented
}
