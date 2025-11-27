package file

import (
	"bytes"
	"fmt"
	"github.com/klauspost/compress/zstd"
	"github.com/mogud/snow/core/task"
	"io"
	"log"
	"os"
	"path"
)

type writerElement struct {
	File    string
	Message string
}

type writer struct {
	compress bool
	fileName string
	file     *os.File
}

func newWriter(c <-chan *writerElement, compress bool) *writer {
	w := &writer{
		compress: compress,
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

		if ss.fileName == unit.File {
			_, _ = fmt.Fprintln(ss.file, unit.Message)
			continue
		}

		dir := path.Dir(unit.File)
		_ = os.MkdirAll(dir, 0755)

		f, err := os.OpenFile(unit.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("### ERROR ### log to file <%s>: %s", unit.File, err.Error())
			continue
		}

		if ss.file != nil {
			_ = ss.file.Close()
		}

		log.Printf("### NOTICE ### create log file <%s>", unit.File)
		ss.file = f
		_, _ = fmt.Fprintln(f, unit.Message)

		previousName := ss.fileName

		ss.fileName = unit.File

		if !ss.compress || len(ss.fileName) == 0 {
			continue
		}

		dataToCompress, err := os.ReadFile(previousName)
		if err != nil {
			log.Printf("### ERROR ### compress process read file <%s>: %s", previousName, err.Error())
			continue
		}

		zipFile, err := os.OpenFile(previousName+".zst", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("### ERROR ### compress process create file <%s>: %s", previousName, err.Error())
			continue
		}

		err = compress(bytes.NewReader(dataToCompress), zipFile)
		_ = zipFile.Close()

		if err != nil {
			log.Printf("### ERROR ### compress process <%s>: %s", previousName, err.Error())
			continue
		}

		_ = os.Remove(previousName)
	}
}

func compress(in io.Reader, out io.Writer) error {
	enc, err := zstd.NewWriter(out)
	if err != nil {
		return err
	}
	_, err = io.Copy(enc, in)
	if err != nil {
		_ = enc.Close()
		return err
	}
	return enc.Close()
}
