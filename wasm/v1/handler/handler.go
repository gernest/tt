package handler

import (
	"context"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/gernest/tt/api"
	"github.com/gernest/tt/pkg/zlg"
	"github.com/gernest/tt/wasm"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	proxywasm "mosn.io/proxy-wasm-go-host/proxywasm/v1"
)

type H struct {
	mw          *api.Middleware_Wasm
	vm          *wasm.Wasm
	instance    *wasm.Instance
	log         *zap.Logger
	base        *Wasm
	id          atomic.Int32
	rootContext int32
}

func New(
	ctx context.Context,
	wasmModulesPath string,
	mw *api.Middleware_Wasm,
) (*H, error) {
	file := filepath.Join(wasmModulesPath, mw.Module)
	mwLog := zlg.Logger.Named("PROXY_WASM").With(
		zap.String("middleware", mw.Name),
		zap.String("module", mw.Module),
	)
	mwLog.Info("Compiling wasm module")
	mwLog.Debug("Module path " + file)
	vm := wasm.New(mwLog)
	if err := vm.CompileFile(file); err != nil {
		return nil, err
	}
	var id atomic.Int32
	rootContext := id.Inc()
	mwLog = mwLog.With(
		zap.Int32("rootContext", rootContext),
	)
	mwLog.Info("Creating new wasm instance")
	instance, err := vm.NewInstance(mw)
	if err != nil {
		mwLog.Error("Failed to create wasm instance", zap.Error(err))
		return nil, err
	}
	// we start the module instance beforehand.
	mwLog.Info("Starting wasm module instance")
	err = instance.Start()
	if err != nil {
		mwLog.Error("Failed to start wasm instance", zap.Error(err))
		return nil, err
	}
	base := &Wasm{}
	base.L = mwLog
	bufFn, releaseBuf := safeBuffer()
	base.NewBuffer = bufFn
	runtime.SetFinalizer(base, func() {
		releaseBuf()
	})
	rootABI := &proxywasm.ABIContext{
		Imports:  base,
		Instance: instance,
	}
	export := rootABI.GetExports()
	// create root plugin context
	mwLog.Info("Creating root context")
	err = export.ProxyOnContextCreate(rootContext, 0)
	if err != nil {
		mwLog.Error("Failed creating root context", zap.Error(err))
		return nil, err
	}
	return &H{
		mw:          mw,
		vm:          vm,
		instance:    instance,
		log:         mwLog,
		id:          id,
		base:        base,
		rootContext: rootContext,
	}, nil
}

func (h *H) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// create a http context
		httpContextID := h.id.Inc()
		baseCtx := h.base.Clone()
		baseCtx.Request.Request = r
		mwLog := h.log.With(zap.Int32("httpContextID", httpContextID))
		baseCtx.L = mwLog
		ctxBuf, releaseCTxBuf := safeBuffer()
		// make sure all buffers created in this context are properly released. after
		// the request has been served.
		defer releaseCTxBuf()
		baseCtx.NewBuffer = ctxBuf

		abi := h.abi(baseCtx)
		exports := abi.GetExports()
		ctx := &ExecContext{
			Log:         mwLog,
			ContextID:   httpContextID,
			RootContext: h.rootContext,
			Exports:     exports,
		}
		if err := ctx.Before(); err != nil {
			mwLog.Error("ProxyOnContextCreate", zap.Error(err))
			h.E500(w, r)
			return
		}
		//make sure we destroy the context when we are done
		defer func() {
			if err := ctx.After(); err != nil {
				mwLog.Error("ProxyOnContextFinalize", zap.Error(err))
			}
		}()
		action, err := exports.ProxyOnRequestHeaders(httpContextID, int32(len(r.Header)), 0)
		if err != nil {
			mwLog.Error("ProxyOnHttpRequestHeaders", zap.Error(err))
			return
		}
		mwLog.Info("Responded with", zap.Int32("action", int32(action)))
		next.ServeHTTP(w, r)
	})
}

func (h *H) abi(b *Wasm) *proxywasm.ABIContext {
	return &proxywasm.ABIContext{
		Imports:  b,
		Instance: h.instance,
	}
}

func (h *H) E500(w http.ResponseWriter, r *http.Request) {
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

type ExecContext struct {
	Log         *zap.Logger
	ContextID   int32
	RootContext int32
	Exports     proxywasm.Exports
	Request     *http.Request
	Response    http.ResponseWriter
}

func (e *ExecContext) Before() error {
	return e.Exports.ProxyOnContextCreate(
		e.ContextID, e.RootContext,
	)
}

func (e *ExecContext) Apply() (applyNext bool) {
	return e.apply(
		e.httpRequest()...,
	)
}

func (e *ExecContext) apply(fns ...applyFn) (applyNext bool) {
	for _, fn := range fns {
		action, name, err := fn()
		if err != nil {
			e.Log.Error(name, zap.Error(err))
			return false
		}
		if action != proxywasm.ActionContinue {
			return false
		}
	}
	return true
}

type applyFn func() (action proxywasm.Action, name string, err error)

func (e *ExecContext) After() error {
	_, err := e.Exports.ProxyOnDone(e.ContextID)
	return err
}

func (e *ExecContext) httpRequest() []applyFn {
	return []applyFn{
		func() (action proxywasm.Action, name string, err error) {
			a, err := e.Exports.ProxyOnRequestHeaders(
				e.ContextID, int32(len(e.Request.Header)), 0,
			)
			return a, "ProxyOnHttpRequestHeaders", err
		},
		func() (action proxywasm.Action, name string, err error) {
			a, err := e.Exports.ProxyOnRequestTrailers(
				e.ContextID, int32(len(e.Request.Trailer)),
			)
			return a, "ProxyOnHttpRequestTrailers", err
		},
	}
}
