package file

import (
	"fmt"
	"os"
	"path"
	"snow/core/task"
)

type writerElement struct {
	File    string
	Message string
}

type writer struct {
	fileName string
	file     *os.File
}

func newWriter(c <-chan *writerElement) *writer {
	w := &writer{}
	task.Execute(func() { w.loop(c) })
	return w
}

func (ss *writer) loop(c <-chan *writerElement) {
	for {
		unit, ok := <-c
		if !ok {
			break
		}

		if ss.fileName == unit.File {
			_, _ = fmt.Fprintln(ss.file, unit.Message)
			continue
		}

		dir := path.Dir(unit.File)
		_ = os.MkdirAll(dir, 0755)

		f, err := os.OpenFile(unit.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("### ERROR ### log to file <%s>: %s\n", unit.File, err.Error())
			continue
		}

		if ss.file != nil {
			_ = ss.file.Close()
		}

		fmt.Printf("### NOTICE ### create log file <%s>\n", unit.File)
		ss.file = f
		ss.fileName = unit.File
		_, _ = fmt.Fprintln(f, unit.Message)
	}
}
