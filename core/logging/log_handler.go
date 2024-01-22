package logging

import "time"

type LogData struct {
	Time    time.Time
	Path    string
	Name    string
	ID      string
	File    string
	Line    int
	Level   Level
	Custom  []any
	Message func() string
}

type ILogHandler interface {
	Log(data *LogData)
}
