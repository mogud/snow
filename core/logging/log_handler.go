package logging

import (
	"log"
	"time"
)

type LogData struct {
	Time     time.Time
	NodeID   int
	NodeName string
	Path     string
	Name     string
	ID       string
	File     string
	Line     int
	Level    Level
	Custom   []any
	Message  func() string
}

type ILogHandler interface {
	Log(data *LogData)
}

func NewSimpleLogHandler() ILogHandler {
	return simpleLogHandler{}
}

type simpleLogHandler struct {
}

func (s simpleLogHandler) Log(data *LogData) {
	log.Println(DefaultLogFormatter(data))
}
