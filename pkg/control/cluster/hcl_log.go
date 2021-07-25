package cluster

import (
	"io"
	"log"

	"github.com/hashicorp/go-hclog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ hclog.Logger = (*HCLogAdapter)(nil)

// NewHCLogAdapter creates a new adapter, wrapping an underlying
// zap.Logger inside an implementation that emulates hclog.Logger.
func NewHCLogAdapter(wrapped *zap.Logger) *HCLogAdapter {
	if wrapped == nil {
		wrapped = zap.L()
	}
	wrapped = zap.New(wrapped.Core(), zap.AddCallerSkip(1))
	return &HCLogAdapter{
		name: "",
		zap:  wrapped,
	}
}

// HCLogAdapter is an adapter that allows to use a zap.Logger where
// and hclog.Logger is expected.
type HCLogAdapter struct {
	name string
	zap  *zap.Logger
}

func (l *HCLogAdapter) Clone() *HCLogAdapter {
	return &HCLogAdapter{
		name: l.name,
		zap:  l.zap,
	}
}

// // Logger describes the interface that must be implemeted by all loggers.
// type Logger interface {
// 	// Args are alternating key, val pairs
// 	// keys must be strings
// 	// vals can be any type, but display is implementation specific
// 	// Emit a message and key/value pairs at a provided log level
func (l *HCLogAdapter) Log(level hclog.Level, msg string, args ...interface{}) {
	fields := []zapcore.Field{}
	for i := 0; i < len(args); i += 2 {
		fields = append(fields, zap.Any(args[i].(string), args[i+1]))
	}
	switch level {
	case hclog.Trace, hclog.Debug:
		l.zap.Debug(msg, fields...)
	case hclog.NoLevel, hclog.Info:
		l.zap.Info(msg, fields...)
	case hclog.Warn:
		l.zap.Warn(msg, fields...)
	case hclog.Error:
		l.zap.Error(msg, fields...)
	}
}

// Emit a message and key/value pairs at the TRACE level
func (l *HCLogAdapter) Trace(msg string, args ...interface{}) {
	fields := []zapcore.Field{}
	for i := 0; i < len(args); i += 2 {
		fields = append(fields, zap.Any(args[i].(string), args[i+1]))
	}
	l.zap.Debug(msg, fields...)
}

// Emit a message and key/value pairs at the DEBUG level
func (l *HCLogAdapter) Debug(msg string, args ...interface{}) {
	fields := []zapcore.Field{}
	for i := 0; i < len(args); i += 2 {
		fields = append(fields, zap.Any(args[i].(string), args[i+1]))
	}
	l.zap.Debug(msg, fields...)
}

// Emit a message and key/value pairs at the INFO level
func (l *HCLogAdapter) Info(msg string, args ...interface{}) {
	fields := []zapcore.Field{}
	for i := 0; i < len(args); i += 2 {
		fields = append(fields, zap.Any(args[i].(string), args[i+1]))
	}
	l.zap.Info(msg, fields...)
}

// Emit a message and key/value pairs at the WARN level
func (l *HCLogAdapter) Warn(msg string, args ...interface{}) {
	fields := []zapcore.Field{}
	for i := 0; i < len(args); i += 2 {
		fields = append(fields, zap.Any(args[i].(string), args[i+1]))
	}
	l.zap.Warn(msg, fields...)
}

// Emit a message and key/value pairs at the ERROR level
func (l *HCLogAdapter) Error(msg string, args ...interface{}) {
	fields := []zapcore.Field{}
	for i := 0; i < len(args); i += 2 {
		fields = append(fields, zap.Any(args[i].(string), args[i+1]))
	}
	l.zap.Error(msg, fields...)
}

func (l *HCLogAdapter) IsTrace() bool {
	return l.zap.Core().Enabled(zap.DebugLevel)
}

func (l *HCLogAdapter) IsDebug() bool {
	return l.zap.Core().Enabled(zap.DebugLevel)
}

func (l *HCLogAdapter) IsInfo() bool {
	return l.zap.Core().Enabled(zap.InfoLevel)
}

func (l *HCLogAdapter) IsWarn() bool {
	return l.zap.Core().Enabled(zap.WarnLevel)
}

func (l *HCLogAdapter) IsError() bool {
	return l.zap.Core().Enabled(zap.ErrorLevel)
}

// ImpliedArgs returns With key/value pairs
func (l *HCLogAdapter) ImpliedArgs() []interface{} {
	return nil
}

func (l *HCLogAdapter) With(args ...interface{}) hclog.Logger {
	fields := []zapcore.Field{}
	for i := 0; i < len(args); i += 2 {
		fields = append(fields, zap.Any(args[i].(string), args[i+1]))
	}
	return NewHCLogAdapter(l.zap.With(fields...))
}

func (l *HCLogAdapter) Name() string {
	return l.name
}

func (l *HCLogAdapter) Named(name string) hclog.Logger {
	nl := l.Clone()
	nl.name = nl.name + "name"
	return nl
}

func (l *HCLogAdapter) ResetNamed(name string) hclog.Logger {
	nl := l.Clone()
	nl.name = name
	return nl
}

func (l *HCLogAdapter) SetLevel(level hclog.Level) {
	switch level {
	case hclog.Trace, hclog.Debug:
	case hclog.NoLevel, hclog.Info:
	case hclog.Warn:
	case hclog.Error:
	}
}

func (l *HCLogAdapter) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	return zap.NewStdLog(l.zap)
}

// Return a value that conforms to io.Writer, which can be passed into log.SetOutput()
func (l *HCLogAdapter) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	return nil
}
