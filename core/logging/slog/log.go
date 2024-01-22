package slog

import (
	"snow/core/logging"
	"snow/core/logging/handler"
	"sync/atomic"
)

var globalHandler logging.ILogHandler
var globalLogger logging.ILogger
var lock int32

func BindGlobalHandler(h logging.ILogHandler) {
	for {
		if !atomic.CompareAndSwapInt32(&lock, 0, 1) {
			continue
		}

		globalHandler = h

		atomic.StoreInt32(&lock, 0)
		break
	}
}

func BindGlobalLogger(l logging.ILogger) {
	for {
		if !atomic.CompareAndSwapInt32(&lock, 0, 1) {
			continue
		}

		globalLogger = l

		atomic.StoreInt32(&lock, 0)
		break
	}
}

func getLogger() logging.ILogger {
	var logger logging.ILogger
	for {
		if !atomic.CompareAndSwapInt32(&lock, 0, 1) {
			continue
		}

		if globalHandler == nil {
			globalHandler = handler.NewCompoundHandler()
		}
		if globalLogger == nil {
			globalLogger = logging.NewDefaultLogger("Global", globalHandler, nil)
		}
		logger = globalLogger

		atomic.StoreInt32(&lock, 0)
		break
	}
	return logger
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
