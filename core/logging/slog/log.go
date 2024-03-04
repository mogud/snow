package slog

import (
	"gitee.com/mogud/snow/core/logging"
	"sync/atomic"
)

var globalLogger atomic.Pointer[logging.ILogger]

func init() {
	logger := logging.ILogger(logging.NewDefaultLogger("Global", logging.NewSimpleLogHandler(), nil))
	globalLogger.Store(&logger)
}

func BindGlobalHandler(h logging.ILogHandler) {
	logger := logging.ILogger(logging.NewDefaultLogger("Global", h, nil))
	globalLogger.Store(&logger)
}

func BindGlobalLogger(l logging.ILogger) {
	if l == nil {
		panic("bind global logger with nil")
	}

	globalLogger.Store(&l)
}

func getLogger() logging.ILogger {
	return *globalLogger.Load()
}

func Tracef(format string, args ...any) {
	getLogger().Tracef(format, args...)
}

func Debugf(format string, args ...any) {
	getLogger().Debugf(format, args...)
}

func Infof(format string, args ...any) {
	getLogger().Infof(format, args...)
}

func Warnf(format string, args ...any) {
	getLogger().Warnf(format, args...)
}

func Errorf(format string, args ...any) {
	getLogger().Errorf(format, args...)
}

func Fatalf(format string, args ...any) {
	getLogger().Fatalf(format, args...)
}
