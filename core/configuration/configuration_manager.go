package configuration

import (
	"snow/core/container"
	"snow/core/notifier"
	"sync"
)

var _ IConfigurationManager = (*Manager)(nil)

type Manager struct {
	lock       sync.Mutex
	properties container.Map[string, any]
	sources    container.List[IConfigurationSource]
	providers  container.List[IConfigurationProvider]
	notifier   *Notifier
}

func NewManager() *Manager {
	return &Manager{
		properties: make(container.Map[string, any]),
		sources:    container.NewList[IConfigurationSource](),
		providers:  container.NewList[IConfigurationProvider](),
		notifier:   NewNotifier(),
	}
}

func (ss *Manager) Get(key string) string {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	for i := ss.providers.Len() - 1; i >= 0; i-- {
		if value, ok := ss.providers[i].TryGet(key); ok {
			return value
		}
	}
	return ""
}

func (ss *Manager) TryGet(key string) (value string, ok bool) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	for i := ss.providers.Len() - 1; i >= 0; i-- {
		if value, ok = ss.providers[i].TryGet(key); ok {
			return value, true
		}
	}
	return
}

func (ss *Manager) Set(key string, value string) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	for _, provider := range ss.providers {
		provider.Set(key, value)
	}
}

func (ss *Manager) GetSection(key string) IConfigurationSection {
	return NewSection(ss, key)
}

func (ss *Manager) GetChildren() container.List[IConfigurationSection] {
	return ss.GetChildrenByPath("")
}

func (ss *Manager) GetChildrenByPath(path string) container.List[IConfigurationSection] {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	keySet := container.NewSet[string]()
	for _, provider := range ss.providers {
		for _, key := range provider.GetChildKeys(path) {
			keySet.Add(key)
		}
	}

	sections := make([]IConfigurationSection, 0, keySet.Len())
	if len(path) > 0 {
		for key := range keySet {
			sections = append(sections, ss.GetSection(path+KeyDelimiter+key))
		}
	} else {
		for key := range keySet {
			sections = append(sections, ss.GetSection(key))
		}
	}
	return sections
}

func (ss *Manager) GetReloadNotifier() notifier.INotifier {
	return ss.notifier
}

func (ss *Manager) Reload() {
	defer ss.notifier.Notify()

	ss.lock.Lock()
	defer ss.lock.Unlock()

	for _, provider := range ss.providers {
		provider.Load()
	}
}

func (ss *Manager) GetProviders() container.List[IConfigurationProvider] {
	return ss.providers
}

func (ss *Manager) GetProperties() container.Map[string, any] {
	return ss.properties
}

func (ss *Manager) GetSources() container.List[IConfigurationSource] {
	return ss.sources
}

func (ss *Manager) AddSource(source IConfigurationSource) {
	newProvider := source.BuildConfigurationProvider(ss)
	newProvider.Load()

	defer ss.notifier.Notify()

	ss.lock.Lock()
	defer ss.lock.Unlock()

	ss.sources.Add(source)
	ss.providers.Add(newProvider)

	newProvider.GetReloadNotifier().RegisterNotifyCallback(ss.notifier.Notify)
}

func (ss *Manager) BuildConfigurationRoot() IConfigurationRoot {
	return ss
}
