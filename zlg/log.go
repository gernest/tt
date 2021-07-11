package zlg

import (
	"context"

	"go.uber.org/zap"
)

// Logger global default logger
var Logger *zap.Logger
var Level zap.AtomicLevel

func init() {
	var err error
	c := zap.NewProductionConfig()
	c.DisableStacktrace = true
	c.Level.SetLevel(zap.DebugLevel)
	Logger, err = c.Build(
		zap.WithCaller(false),
	)
	if err != nil {
		panic(err)
	}
	Level = c.Level
}

// Info logs info
func Info(msg string, f ...zap.Field) {
	Logger.Info(msg, f...)
}

func Debug(msg string, f ...zap.Field) {
	Logger.Debug(msg, f...)
}

func Error(err error, msg string, f ...zap.Field) {
	Logger.Error(msg, append(f, zap.Error(err))...)
}

type zapKey struct{}

func Set(ctx context.Context, lg *zap.Logger) context.Context {
	return context.WithValue(ctx, zapKey{}, lg)
}

func Get(ctx context.Context) *zap.Logger {
	if v := ctx.Value(zapKey{}); v != nil {
		return v.(*zap.Logger)
	}
	return Logger
}
