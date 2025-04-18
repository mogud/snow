package configuration

import (
	"github.com/mogud/snow/core/container"
	"github.com/mogud/snow/core/notifier"
	"sync"
)

var _ IConfigurationRoot = (*Root)(nil)

type Root struct {
	lock sync.Mutex

	providers container.List[IConfigurationProvider]
	notifier  *Notifier
}

func NewConfigurationRoot(providers container.List[IConfigurationProvider]) IConfigurationRoot {
	root := &Root{
		providers: providers,
		notifier:  NewNotifier(),
	}

	for _, provider := range providers {
		provider.GetReloadNotifier().RegisterNotifyCallback(root.notifier.Notify)
		provider.Load()
	}

	return root
}

func (ss *Root) Get(key string) string {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	for i := ss.providers.Len() - 1; i >= 0; i-- {
		if value, ok := ss.providers[i].TryGet(key); ok {
			return value
		}
	}
	return ""
}

func (ss *Root) TryGet(key string) (value string, ok bool) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	for i := ss.providers.Len() - 1; i >= 0; i-- {
		if value, ok := ss.providers[i].TryGet(key); ok {
			return value, true
		}
	}
	return "", false
}

func (ss *Root) Set(key string, value string) {
	ss.lock.Lock()
	defer ss.lock.Unlock()

	for _, provider := range ss.providers {
		provider.Set(key, value)
	}
}

func (ss *Root) GetSection(key string) IConfigurationSection {
	return NewSection(ss, key)
}

func (ss *Root) GetChildren() container.List[IConfigurationSection] {
	return ss.GetChildrenByPath("")
}

func (ss *Root) GetChildrenByPath(path string) container.List[IConfigurationSection] {
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

func (ss *Root) GetReloadNotifier() notifier.INotifier {
	return ss.notifier
}

func (ss *Root) Reload() {
	for _, provider := range ss.providers {
		provider.Load()
	}
	ss.notifier.Notify()
}

func (ss *Root) GetProviders() container.List[IConfigurationProvider] {
	return ss.providers
}
