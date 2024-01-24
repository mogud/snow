package file

import (
	"fmt"
	"gitee.com/mogud/snow/logging"
	"gitee.com/mogud/snow/option"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

var _ logging.ILogHandler = (*Handler)(nil)

type Option struct {
	LogPath          string                   `koanf:"LogPath"`
	MaxLogChanLength int                      `koanf:"MaxLogChanLength"`
	Formatter        string                   `koanf:"Formatter"`
	WithFileLine     bool                     `koanf:"WithFileLine"`
	FileLineSkip     int                      `koanf:"FileLineSkip"`
	Filter           map[string]logging.Level `koanf:"Filter"`
}

type Handler struct {
	lock sync.Mutex

	fileName   string
	year       int
	month      time.Month
	day        int
	logChan    chan *writerElement
	fileWriter *writer

	option           *Option
	sortedFilterKeys []string
	formatter        func(logData *logging.LogData) string
}

func NewHandler() *Handler {
	handler := &Handler{
		option: &Option{
			LogPath:          "logs",
			MaxLogChanLength: 102400,
			Formatter:        "Default",
			WithFileLine:     true,
			FileLineSkip:     4,
			Filter:           make(map[string]logging.Level),
		},
		formatter: logging.ColorLogFormatter,
		logChan:   make(chan *writerElement, 102400),
	}
	handler.fileWriter = newWriter(handler.logChan)

	for sPath := range handler.option.Filter {
		handler.sortedFilterKeys = append(handler.sortedFilterKeys, sPath)
	}

	sort.Strings(handler.sortedFilterKeys)
	return handler
}

func (ss *Handler) Construct(option *option.Option[*Option], repo *logging.LogFormatterContainer) {
	ss.option = option.Get()

	formatterName := ss.option.Formatter
	ss.formatter = repo.GetFormatter(formatterName)
	if ss.formatter == nil {
		ss.formatter = logging.DefaultLogFormatter
	}

	if ss.option.MaxLogChanLength != 0 && ss.option.MaxLogChanLength != 102400 {
		close(ss.logChan)
		ss.logChan = make(chan *writerElement, ss.option.MaxLogChanLength)
		ss.fileWriter = newWriter(ss.logChan)
	}

	if len(ss.option.LogPath) == 0 {
		ss.option.LogPath = "logs"
	}
}

func (ss *Handler) Log(logData *logging.LogData) {
	if logData.Level == logging.NONE {
		return
	}

	filterLevel := logging.NONE
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

	ss.refreshFileName(logData.Time)

	unit := &writerElement{
		File:    ss.fileName,
		Message: message,
	}
	select {
	case ss.logChan <- unit:
	default:
		_, _ = fmt.Fprintln(os.Stderr, "file log channel full")
	}
}

func (ss *Handler) refreshFileName(now time.Time) {
	year, month, day := now.Date()

	ss.lock.Lock()
	defer ss.lock.Unlock()
	if len(ss.fileName) == 0 || ss.year != year || ss.month != month || ss.day != day {
		// 跨天了
		ss.year = year
		ss.month = month
		ss.day = day
		ss.fileName = path.Join(ss.option.LogPath, fmt.Sprintf("%04d_%02d_%02d.log", year, month, day))
	}
}
