package file

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"snow/core/logging"
	"snow/core/maps"
	"snow/core/option"
	"sort"
	"strings"
	"sync"
	"time"
)

var _ logging.ILogHandler = (*Handler)(nil)

type Option struct {
	LogPath                    string                   `snow:"LogPath"`
	MaxLogChanLength           int                      `snow:"MaxLogChanLength"`
	Formatter                  string                   `snow:"Formatter"`
	FileLineLevel              int                      `snow:"FileLineLevel"`
	FileLineSkip               int                      `snow:"FileLineSkip"`
	Filter                     map[string]logging.Level `snow:"Filter"`
	DefaultLevel               logging.Level            `snow:"DefaultLevel"`
	FileNameFormat             string                   `snow:"FileNameFormat"`
	FileRollingMegabytes       int                      `snow:"FileRollingMegabytes"`
	FileRollingIntervalSeconds int                      `snow:"FileRollingIntervalSeconds"`
}

type Handler struct {
	lock sync.Mutex

	lastFileLogTime         time.Time
	lastFileNameRefreshTime time.Time
	fileName                string
	fileNameTemplate        string
	index                   int32
	logChan                 chan *writerElement
	fileWriter              *writer

	option            *Option
	sortedFilterKeys  []string
	cacheFilterKeyMap map[string]struct{}
	formatter         func(logData *logging.LogData) string
}

func NewHandler() *Handler {
	handler := &Handler{
		option: &Option{
			LogPath:          "logs",
			MaxLogChanLength: 102400,
			Formatter:        "Default",
			FileLineLevel:    4,
			FileLineSkip:     6,
			Filter:           make(map[string]logging.Level),
		},
		cacheFilterKeyMap: make(map[string]struct{}),
		formatter:         logging.ColorLogFormatter,
		logChan:           make(chan *writerElement, 102400),
	}
	handler.fileWriter = newWriter(handler.logChan)

	for sPath := range handler.option.Filter {
		handler.sortedFilterKeys = append(handler.sortedFilterKeys, sPath)
		handler.cacheFilterKeyMap[sPath] = struct{}{}
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

	ss.CheckOption()

	option.OnChanged(func() {
		newOption := option.Get()

		ss.lock.Lock()
		defer ss.lock.Unlock()

		ss.option = newOption
		ss.CheckOption()
	})
}

func (ss *Handler) CheckOption() {
	ss.sortedFilterKeys = maps.Keys(ss.option.Filter)
	sort.Strings(ss.sortedFilterKeys)

	if ss.option.MaxLogChanLength <= 0 {
		ss.option.MaxLogChanLength = 102400
	}

	if ss.option.FileRollingMegabytes <= 0 {
		ss.option.FileRollingMegabytes = 100
	}

	if ss.option.FileRollingIntervalSeconds <= 0 {
		ss.option.FileRollingIntervalSeconds = 3600
	} else if ss.option.FileRollingIntervalSeconds < 60 {
		ss.option.FileRollingIntervalSeconds = 60
	}

	if len(ss.option.FileNameFormat) == 0 {
		ss.option.FileNameFormat = "%Y_%02M_%02D_%02h_%02m_%04i.log"
	}

	if ss.option.MaxLogChanLength != cap(ss.logChan) {
		close(ss.logChan)
		ss.logChan = make(chan *writerElement, ss.option.MaxLogChanLength)
		ss.fileWriter = newWriter(ss.logChan)
	}

	if len(ss.option.LogPath) == 0 {
		ss.option.LogPath = "logs"
	}
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
	logCh := ss.logChan
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

	fileName := ss.refreshFileName(logData.Time)

	unit := &writerElement{
		File:    fileName,
		Message: message,
	}
	select {
	case logCh <- unit:
	default:
		_, _ = fmt.Fprintln(os.Stderr, "file log channel full")
	}
}

func (ss *Handler) refreshFileName(now time.Time) string {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	if len(ss.fileName) == 0 || now.Sub(ss.lastFileNameRefreshTime) > 10*time.Second {
		ss.lastFileNameRefreshTime = now
		rollingFile := false
		if now.Sub(ss.lastFileLogTime) > time.Duration(ss.option.FileRollingIntervalSeconds)*time.Second {
			ss.lastFileLogTime = now.Truncate(time.Duration(ss.option.FileRollingIntervalSeconds) * time.Second)
			rollingFile = true
		}
		year, month, day := ss.lastFileLogTime.Date()

		escape := false
		indexFormatString := ""
		fileNameTemplateBuilder := &strings.Builder{}
		itemFormatBuilder := &strings.Builder{}
		for _, c := range ss.option.FileNameFormat {
			if escape {
				switch {
				case c == '%':
					fileNameTemplateBuilder.WriteByte('%')
				case c >= '0' && c <= '9':
					itemFormatBuilder.WriteRune(c)
					continue
				case c == 'Y':
					itemFormatBuilder.WriteByte('d')
					fileNameTemplateBuilder.WriteString(fmt.Sprintf(itemFormatBuilder.String(), year))
				case c == 'M':
					itemFormatBuilder.WriteByte('d')
					fileNameTemplateBuilder.WriteString(fmt.Sprintf(itemFormatBuilder.String(), month))
				case c == 'D':
					itemFormatBuilder.WriteByte('d')
					fileNameTemplateBuilder.WriteString(fmt.Sprintf(itemFormatBuilder.String(), day))
				case c == 'h':
					itemFormatBuilder.WriteByte('d')
					fileNameTemplateBuilder.WriteString(fmt.Sprintf(itemFormatBuilder.String(), ss.lastFileLogTime.Hour()))
				case c == 'm':
					itemFormatBuilder.WriteByte('d')
					fileNameTemplateBuilder.WriteString(fmt.Sprintf(itemFormatBuilder.String(), ss.lastFileLogTime.Minute()))
				case c == 'i':
					itemFormatBuilder.WriteByte('d')
					indexFormatString = itemFormatBuilder.String()
					fileNameTemplateBuilder.WriteString("///__INDEX__///")
				default:
					fileNameTemplateBuilder.WriteRune(c)
				}

				itemFormatBuilder.Reset()
				escape = false
				continue
			}

			if c == '%' {
				escape = true
				itemFormatBuilder.WriteByte('%')
				continue
			} else {
				fileNameTemplateBuilder.WriteRune(c)
			}
		}

		newFileNameTemplate := fileNameTemplateBuilder.String()
		if ss.fileNameTemplate != newFileNameTemplate {
			ss.fileNameTemplate = newFileNameTemplate
			ss.index = 0

			baseName := strings.ReplaceAll(ss.fileNameTemplate, "///__INDEX__///", fmt.Sprintf(indexFormatString, ss.index))
			fullName := path.Join(ss.option.LogPath, baseName)
			if len(ss.fileName) == 0 && len(indexFormatString) > 0 {
				for {
					if _, err := os.Stat(fullName); err != nil {
						break
					}

					ss.index++
					baseName = strings.ReplaceAll(ss.fileNameTemplate, "///__INDEX__///", fmt.Sprintf(indexFormatString, ss.index))
					fullName = path.Join(ss.option.LogPath, baseName)
				}
			}
			ss.fileName = fullName
		} else {
			// 此时有两种情况：
			// 1. 模板名因为时间没有变化而不变，但文件名中存在自动索引号，且文件大小超过限制，此时需增加 index
			// 2. 模板名因为时间粒度大于滚动时间粒度而不变，此时需增加 index
			if !rollingFile && len(indexFormatString) > 0 {
				if stat, err := os.Stat(ss.fileName); err == nil && stat.Size() > int64(ss.option.FileRollingMegabytes)*1024*1024 {
					rollingFile = true
				}
			}

			if rollingFile {
				ss.index++
				baseName := strings.ReplaceAll(ss.fileNameTemplate, "///__INDEX__///", fmt.Sprintf(indexFormatString, ss.index))
				ss.fileName = path.Join(ss.option.LogPath, baseName)
			}
		}
	}
	return ss.fileName
}
