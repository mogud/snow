package logging

import "reflect"

type ILoggerInjector interface {
	loggerInjectorTag()
}

var _ ILoggerInjector = (*Logger[int])(nil)

// Logger 用于注入 logger
//
//	T 注入时 path 即 T 的完整路径
type Logger[T any] struct {
	handler ILogHandler
}

func (ss *Logger[T]) Get(logDataBuilder func(data *LogData)) ILogger {
	ty := reflect.TypeOf((*T)(nil)).Elem()
	path := ty.PkgPath() + "/" + ty.Name()
	return NewDefaultLogger(path, ss.handler, logDataBuilder)
}

func (ss *Logger[T]) loggerInjectorTag() {
}
