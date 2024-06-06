package sources

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/mogud/snow/core/configuration"
	"github.com/mogud/snow/core/container"
	"log"
	"os"
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
						go func() {
							time.After(500 * time.Millisecond)
							ss.loadFile()
							ss.OnReload()
						}()
					} else if event.Has(fsnotify.Remove) {
						ss.Replace(container.NewMap[string, string]())
						ss.OnReload()
					}
				case err, ok := <-watcher.Errors:
					log.Println("error:", err)
					if !ok {
						return
					}
				}
			}
		}()

		err = watcher.Add(ss.path)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (ss *FileConfigurationProvider) loadFile() {
	data, err := os.ReadFile(ss.path)
	if err != nil {
		fmt.Printf("read file error: %v\n", err.Error())
		return
	}

	ss.OnLoad(data)
}
