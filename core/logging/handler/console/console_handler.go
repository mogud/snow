package console

import (
	"fmt"
	"os"
	"runtime"
	"snow/core/logging"
	"snow/core/maps"
	"snow/core/option"
	"sort"
	"strings"
	"sync"
)

var _ logging.ILogHandler = (*Handler)(nil)

type Option struct {
	Formatter     string                   `snow:"Formatter"`
	FileLineLevel int                      `snow:"FileLineLevel"`
	FileLineSkip  int                      `snow:"FileLineSkip"`
	ErrorLevel    logging.Level            `snow:"ErrorLevel"`
	Filter        map[string]logging.Level `snow:"Filter"`
	DefaultLevel  logging.Level            `snow:"DefaultLevel"`
}

type Handler struct {
	lock             sync.Mutex
	option           *Option
	sortedFilterKeys []string
	formatter        func(logData *logging.LogData) string
}

func NewHandler() *Handler {
	handler := &Handler{
		option: &Option{
			Formatter:     "Color",
			FileLineLevel: 4,
			FileLineSkip:  6,
			ErrorLevel:    logging.ERROR,
			Filter:        make(map[string]logging.Level),
		},
		formatter: logging.ColorLogFormatter,
	}

	handler.sortedFilterKeys = maps.Keys(handler.option.Filter)
	sort.Strings(handler.sortedFilterKeys)
	return handler
}

func (ss *Handler) Construct(opt *option.Option[*Option], repo *logging.LogFormatterContainer) {
	ss.option = opt.Get()
	formatterName := ss.option.Formatter
	ss.formatter = repo.GetFormatter(formatterName)
	if ss.formatter == nil {
		ss.formatter = logging.ColorLogFormatter
	}
	ss.CheckOption()

	opt.OnChanged(func() {
		newOption := opt.Get()

		ss.lock.Lock()
		defer ss.lock.Unlock()

		ss.option = newOption
		ss.CheckOption()
	})
}

func (ss *Handler) CheckOption() {
	ss.sortedFilterKeys = maps.Keys(ss.option.Filter)
	sort.Strings(ss.sortedFilterKeys)

	if ss.option.DefaultLevel == logging.NONE {
		ss.option.DefaultLevel = logging.INFO
	}
}

func (ss *Handler) Log(logData *logging.LogData) {
	if logData.Level == logging.NONE {
		return
	}

	ss.lock.Lock()
	curOption := ss.option
	filterKeys := ss.sortedFilterKeys
	formatter := ss.formatter
	ss.lock.Unlock()

	filterLevel := curOption.DefaultLevel
	for _, key := range filterKeys {
		if strings.HasPrefix(logData.Path, key) {
			filterLevel = curOption.Filter[key]
			break
		}
	}

	if logData.Level < filterLevel {
		return
	}

	if len(logData.File) == 0 && int(logData.Level) >= curOption.FileLineLevel {
		_, fn, ln, _ := runtime.Caller(curOption.FileLineSkip)
		d := logData
		logData = &logging.LogData{
			Time:     d.Time,
			NodeID:   d.NodeID,
			NodeName: d.NodeName,
			Path:     d.Path,
			Name:     d.Name,
			ID:       d.ID,
			File:     fn,
			Line:     ln,
			Level:    d.Level,
			Custom:   d.Custom,
			Message:  d.Message,
		}
	}

	message := formatter(logData)

	if logData.Level < curOption.ErrorLevel {
		_, _ = fmt.Fprintln(os.Stdout, message)
	} else {
		_, _ = fmt.Fprintln(os.Stderr, message)
	}
}
