package console

import (
	"fmt"
	"github.com/mogud/snow/core/logging"
	"github.com/mogud/snow/core/option"
	"os"
	"runtime"
	"sort"
	"strings"
)

var _ logging.ILogHandler = (*Handler)(nil)

type Option struct {
	Formatter       string                   `koanf:"Formatter"`
	WithFileLine    bool                     `koanf:"WithFileLine"`
	FileLineSkip    int                      `koanf:"FileLineSkip"`
	ErrorLevel      logging.Level            `koanf:"ErrorLevel"`
	Filter          map[string]logging.Level `koanf:"Filter"`
	DefaultLogLevel logging.Level            `koanf:"DefaultLogLevel"`
}

type Handler struct {
	option           *Option
	sortedFilterKeys []string
	formatter        func(logData *logging.LogData) string
}

func NewHandler() *Handler {
	handler := &Handler{
		option: &Option{
			Formatter:    "Color",
			WithFileLine: true,
			FileLineSkip: 4,
			ErrorLevel:   logging.ERROR,
			Filter:       make(map[string]logging.Level),
		},
		formatter: logging.ColorLogFormatter,
	}

	for path := range handler.option.Filter {
		handler.sortedFilterKeys = append(handler.sortedFilterKeys, path)
	}

	sort.Strings(handler.sortedFilterKeys)
	return handler
}

func (ss *Handler) Construct(option *option.Option[*Option], repo *logging.LogFormatterContainer) {
	ss.option = option.Get()

	formatterName := ss.option.Formatter
	ss.formatter = repo.GetFormatter(formatterName)
	if ss.formatter == nil {
		ss.formatter = logging.ColorLogFormatter
	}
	if ss.option.DefaultLogLevel == logging.NONE {
		ss.option.DefaultLogLevel = logging.INFO
	}
}

func (ss *Handler) Log(logData *logging.LogData) {
	if logData.Level == logging.NONE {
		return
	}

	filterLevel := ss.option.DefaultLogLevel
	for _, key := range ss.sortedFilterKeys {
		if strings.HasPrefix(logData.Path, key) {
			filterLevel = ss.option.Filter[key]
			break
		}
	}

	if logData.Level < filterLevel {
		return
	}

	if len(logData.File) == 0 && ss.option.WithFileLine {
		_, fn, ln, _ := runtime.Caller(ss.option.FileLineSkip)
		d := logData
		logData = &logging.LogData{
			Time:    d.Time,
			Path:    d.Path,
			Name:    d.Name,
			ID:      d.ID,
			File:    fn,
			Line:    ln,
			Level:   d.Level,
			Custom:  d.Custom,
			Message: d.Message,
		}
	}

	message := ss.formatter(logData)

	if logData.Level < ss.option.ErrorLevel {
		_, _ = fmt.Fprintln(os.Stdout, message)
	} else {
		_, _ = fmt.Fprintln(os.Stderr, message)
	}
}
