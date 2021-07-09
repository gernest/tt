package xwasm

import (
	"context"
	"net/http"

	"github.com/gernest/tt/api"
	"github.com/gernest/tt/wasm"
	"github.com/gernest/tt/zlg"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	proxywasm "mosn.io/proxy-wasm-go-host/proxywasm/v2"
)

// New creates a new instance of the module to be executed as a middleware and
// returns the wasm abi context.
func New(
	ctx context.Context,
	vm *wasm.Wasm,
	mw *api.Middleware_Wasm,
) (*proxywasm.ABIContext, error) {
	inst, err := vm.NewInstance(mw)
	if err != nil {
		return nil, err
	}
	return &proxywasm.ABIContext{
		Instance: inst,
		Imports:  NewImports(ctx, vm, mw, inst),
	}, nil
}

func NewImports(
	ctx context.Context,
	vm *wasm.Wasm,
	mw *api.Middleware_Wasm,
	inst *wasm.Instance,
) *Wasm {
	return &Wasm{}
}

func Handler(ctx context.Context,
	vm *wasm.Wasm,
	mw *api.Middleware_Wasm) (func(http.Handler) http.Handler, error) {

	var id atomic.Int32
	rootContext := id.Inc()
	mwLog := zlg.Logger.Named("PROXY_WASM").With(
		zap.String("middleware", mw.Name),
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
	base := NewImports(ctx, vm, mw, instance)
	rootABI := &proxywasm.ABIContext{
		Imports:  base,
		Instance: instance,
	}
	export := rootABI.GetExports()
	// create root plugin context
	mwLog.Info("Creating root context")
	_, err = export.ProxyOnContextCreate(rootContext, 0, proxywasm.ContextTypePluginContext)
	if err != nil {
		mwLog.Error("Failed creating root context", zap.Error(err))
		return nil, err
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// create a http context
			httpContextID := id.Inc()
			baseCtx := base.Clone()
			baseCtx.Request.Request = r
			mwLog = mwLog.With(zap.Int32("httpContextID", httpContextID))
			baseCtx.L = mwLog
			activeABI := &proxywasm.ABIContext{
				Imports:  baseCtx,
				Instance: instance,
			}
			activeExports := activeABI.GetExports()
			_, err := activeExports.ProxyOnContextCreate(
				httpContextID, rootContext, proxywasm.ContextTypeHttpContext,
			)
			if err != nil {
				mwLog.Error("ProxyOnContextCreate", zap.Error(err))
			}
			//make sure we destroy the context when we are done
			defer func() {
				_, err := activeExports.ProxyOnContextFinalize(httpContextID)
				if err != nil {
					mwLog.Error("ProxyOnContextFinalize", zap.Error(err))
				}
			}()
			activeExports.ProxyOnHttpRequestHeaders()

			if mw.Order == api.Middleware_PRE {
				// we are applying the module before applying the next handler. This means
				// all hooks which are related to response will not apply here because we
				// don't have the response yet.
				next.ServeHTTP(w, r)
				return
			}
			// first we servefinal http Handler then we apply the middleware that way we
			// have access to both the request and the response within this context.
			next.ServeHTTP(w, r)
		})
	}, nil
}
