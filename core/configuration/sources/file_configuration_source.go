package sources

import (
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/mogud/snow/core/configuration"
	"github.com/mogud/snow/core/container"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

var _ configuration.IConfigurationSource = (*FileConfigurationSource)(nil)

type FileConfigurationSource struct {
	Path           string
	Optional       bool
	ReloadOnChange bool
}

func (ss *FileConfigurationSource) BuildConfigurationProvider(builder configuration.IConfigurationBuilder) configuration.IConfigurationProvider {
	return NewFileConfigurationProvider(ss)
}

var _ configuration.IConfigurationProvider = (*FileConfigurationProvider)(nil)

type FileConfigurationProvider struct {
	*configuration.Provider

	path           string
	optional       bool
	reloadOnChange bool
	loaded         bool

	OnLoad func(bytes []byte)
}

func NewFileConfigurationProvider(source *FileConfigurationSource) *FileConfigurationProvider {
	return &FileConfigurationProvider{
		Provider:       configuration.NewProvider(),
		path:           source.Path,
		optional:       source.Optional,
		reloadOnChange: source.ReloadOnChange,
		OnLoad:         func(bytes []byte) {},
	}
}

func (ss *FileConfigurationProvider) Load() {
	if ss.loaded {
		if ss.reloadOnChange {
			return
		}

		ss.loadFile()
		ss.OnReload()
		return
	}
	ss.loadFile()
	ss.loaded = true

	if ss.reloadOnChange {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Fatal(err)
		}

		go func() {
			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					if event.Has(fsnotify.Write | fsnotify.Create) {
						if pathEquals(event.Name, ss.path) {
							log.Printf("file watcher received Write Or Create: %v", event.Name)
							go func() {
								time.After(500 * time.Millisecond)
								ss.loadFile()
								ss.OnReload()
							}()
						}
					} else if event.Has(fsnotify.Rename | fsnotify.Remove) {
						if pathEquals(event.Name, ss.path) {
							log.Printf("file watcher received Rename or Remove: %v", event.Name)
							ss.Replace(container.NewMap[string, string]())
							ss.OnReload()
						}
					}
				case err, ok := <-watcher.Errors:
					log.Println("file watcher error:", err)
					if !ok {
						return
					}
				}
			}
		}()

		err = watcher.Add(ss.path)
		if err != nil {
			parentDir := path.Dir(ss.path)

			log.Printf("file watcher watch not exist file(%v): %v, try to watch parent(%v)...", ss.path, err.Error(), parentDir)

			err = os.MkdirAll(parentDir, os.ModeDir)
			if err != nil {
				log.Fatalf("create watcher path(%v) failed: %v", parentDir, err.Error())
			}
			err = watcher.Add(parentDir)
			if err != nil {
				log.Fatalf("cannot watch path(%v): %v", parentDir, err.Error())
			}
		}
	}
}

func (ss *FileConfigurationProvider) loadFile() {
	if !ss.optional {
		if _, err := os.Stat(ss.path); errors.Is(err, os.ErrNotExist) {
			panic(fmt.Sprintf("file not found: %v", ss.path))
		}
	}

	data, err := os.ReadFile(ss.path)
	if err != nil {
		fmt.Printf("read file error: %v\n", err.Error())
		return
	}

	ss.OnLoad(data)
}

func pathEquals(path1, path2 string) bool {
	p1, p2 := strings.Replace(path1, "\\", "/", -1), strings.Replace(path2, "\\", "/", -1)
	return p1 == p2
}
