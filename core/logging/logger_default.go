package logging

import (
	"fmt"
	"time"
)

var _ ILogger = (*DefaultLogger)(nil)

type DefaultLogger struct {
	path           string
	handler        ILogHandler
	logDataBuilder func(data *LogData)
}

func NewDefaultLogger(path string, handler ILogHandler, logDataBuilder func(data *LogData)) *DefaultLogger {
	return &DefaultLogger{
		path:           path,
		handler:        handler,
		logDataBuilder: logDataBuilder,
	}
}

func (ss *DefaultLogger) log(level Level, format string, args ...any) {
	logData := &LogData{
		Time:  time.Now(),
		Path:  ss.path,
		Level: level,
		Message: func() string {
			return fmt.Sprintf(format, args...)
		},
	}
	if ss.logDataBuilder != nil {
		ss.logDataBuilder(logData)
	}
	ss.handler.Log(logData)
}

func (ss *DefaultLogger) Tracef(format string, args ...any) {
	ss.log(TRACE, format, args...)
}

func (ss *DefaultLogger) Debugf(format string, args ...any) {
	ss.log(DEBUG, format, args...)
}

func (ss *DefaultLogger) Infof(format string, args ...any) {
	ss.log(INFO, format, args...)
}

func (ss *DefaultLogger) Warnf(format string, args ...any) {
	ss.log(WARN, format, args...)
}

func (ss *DefaultLogger) Errorf(format string, args ...any) {
	ss.log(ERROR, format, args...)
}

func (ss *DefaultLogger) Fatalf(format string, args ...any) {
	ss.log(FATAL, format, args...)
}
