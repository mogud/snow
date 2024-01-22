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
	files map[string]*os.File
}

func newWriter(c <-chan *writerElement) *writer {
	w := &writer{
		files: make(map[string]*os.File),
	}
	task.Execute(func() { w.loop(c) })
	return w
}

func (ss *writer) loop(c <-chan *writerElement) {
	for {
		unit, ok := <-c
		if !ok {
			break
		}

		if unit == nil { // clear files
			for _, f := range ss.files {
				_ = f.Close()
			}
			ss.files = make(map[string]*os.File)
			continue
		}

		if f, ok := ss.files[unit.File]; ok {
			_, _ = fmt.Fprintln(f, unit.Message)
			continue
		}

		dir := path.Dir(unit.File)
		_ = os.MkdirAll(dir, 0755)

		f, err := os.OpenFile(unit.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("### ERROR ### log to file <%s>: %s\n", unit.File, err.Error())
			continue
		}

		ss.files[unit.File] = f
		_, _ = fmt.Fprintln(f, unit.Message)
	}
}
