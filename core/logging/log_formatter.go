package logging

import (
	"fmt"
	"strings"
)

type LogFormatterContainer struct {
	formatters map[string]func(logData *LogData) string
}

func NewLogFormatterRepository() *LogFormatterContainer {
	return &LogFormatterContainer{
		formatters: make(map[string]func(logData *LogData) string),
	}
}

func (ss *LogFormatterContainer) AddFormatter(name string, formatter func(logData *LogData) string) {
	ss.formatters[name] = formatter
}

func (ss *LogFormatterContainer) GetFormatter(name string) func(logData *LogData) string {
	return ss.formatters[name]
}

func DefaultLogFormatter(logData *LogData) string {
	sb := strings.Builder{}
	now := logData.Time
	year, mon, day := now.Date()
	hour, m, sec := now.Clock()
	sb.WriteString(fmt.Sprintf(
		"%04d/%02d/%02d %02d:%02d:%02d.%02d",
		year, mon, day,
		hour, m, sec,
		now.Nanosecond()/1000/1000/10,
	))
	if len(logData.NodeName) > 26 {
		sb.WriteString(fmt.Sprintf(" %6d|%-17s..", logData.NodeID, logData.NodeName[:24]))
	} else {
		sb.WriteString(fmt.Sprintf(" %6d|%-19s", logData.NodeID, logData.NodeName))
	}
	sb.WriteString(" ")
	sb.WriteString(l2info[logData.Level].str)

	if len(logData.ID) == 0 {
		sb.WriteString("            -")
	} else if len(logData.ID) > 12 {
		sb.WriteString(fmt.Sprintf(" %10s..", logData.ID[:10]))
	} else {
		sb.WriteString(fmt.Sprintf(" %12s", logData.ID))
	}

	if len(logData.Name) == 0 {
		sb.WriteString("           System")
	} else if len(logData.Name) > 16 {
		sb.WriteString(fmt.Sprintf(" %14s..", logData.Name[:14]))
	} else {
		sb.WriteString(fmt.Sprintf(" %16s", logData.Name))
	}

	if len(logData.File) != 0 {
		sb.WriteString(fmt.Sprintf(" %s(%d)", logData.File, logData.Line))
	}
	sb.WriteString(" ")
	sb.WriteString(logData.Message())
	return sb.String()
}

func ColorLogFormatter(logData *LogData) string {
	sb := strings.Builder{}
	now := logData.Time
	year, mon, day := now.Date()
	hour, m, sec := now.Clock()
	sb.WriteString(fmt.Sprintf(
		"%04d/%02d/%02d %02d:%02d:%02d.%02d",
		year, mon, day,
		hour, m, sec,
		now.Nanosecond()/1000/1000/10,
	))
	if len(logData.NodeName) > 26 {
		sb.WriteString(fmt.Sprintf(" %6d|%-17s..", logData.NodeID, logData.NodeName[:24]))
	} else {
		sb.WriteString(fmt.Sprintf(" %6d|%-19s", logData.NodeID, logData.NodeName))
	}
	sb.WriteString(l2info[logData.Level].color)
	sb.WriteString(" ")
	sb.WriteString(l2info[logData.Level].str)

	if len(logData.ID) == 0 {
		sb.WriteString("            -")
	} else if len(logData.ID) > 12 {
		sb.WriteString(fmt.Sprintf(" %10s..", logData.ID[:10]))
	} else {
		sb.WriteString(fmt.Sprintf(" %12s", logData.ID))
	}

	if len(logData.Name) == 0 {
		sb.WriteString("           System")
	} else if len(logData.Name) > 16 {
		sb.WriteString(fmt.Sprintf(" %14s..", logData.Name[:14]))
	} else {
		sb.WriteString(fmt.Sprintf(" %16s", logData.Name))
	}

	if len(logData.File) != 0 {
		sb.WriteString(fmt.Sprintf(" %s(%d)", logData.File, logData.Line))
	}
	sb.WriteString(" ")
	sb.WriteString(logData.Message())
	sb.WriteString("\x1b[0m")
	return sb.String()
}

type levelInfo struct {
	str   string
	color string
}

var l2info = [...]levelInfo{
	TRACE: {"TRACE", "\x1b[1;34m"},
	DEBUG: {"DEBUG", "\x1b[1;36m"},
	INFO:  {" INFO", "\x1b[1;37m"},
	WARN:  {" WARN", "\x1b[1;33m"},
	ERROR: {"ERROR", "\x1b[1;31m"},
	FATAL: {"FATAL", "\x1b[1;41m"},
}
