package logging

type Level int

const (
	NONE Level = iota
	TRACE
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)
