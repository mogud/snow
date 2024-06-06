package configuration

import (
	"github.com/mogud/snow/core/container"
	"github.com/mogud/snow/core/notifier"
	"strings"
	"sync"
)

var _ IConfigurationProvider = (*Provider)(nil)

type Provider struct {
	lock     sync.Mutex
	data     container.Map[string, string]
	notifier *Notifier
}

func NewProvider() *Provider {
	return &Provider{
		data:     container.NewMap[string, string](),
		notifier: NewNotifier(),
	}
}

func (ss *Provider) Get(key string) string {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	return ss.data[strings.ToUpper(key)]
}

func (ss *Provider) TryGet(key string) (value string, ok bool) {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	value, ok = ss.data[strings.ToUpper(key)]
	return
}

func (ss *Provider) Set(key string, value string) {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	ss.data[strings.ToUpper(key)] = value
}

func (ss *Provider) GetReloadNotifier() notifier.INotifier {
	return ss.notifier
}

func (ss *Provider) Replace(data container.Map[string, string]) {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	ss.data = data
}

func (ss *Provider) Load() {
}

func (ss *Provider) OnReload() {
	ss.notifier.Notify()
}

func (ss *Provider) GetChildKeys(parentPath string) container.List[string] {
	ss.lock.Lock()
	ss.lock.Unlock()

	parentPath = strings.ToUpper(parentPath)
	childKeys := container.NewList[string]()
	if len(parentPath) == 0 {
		for key := range ss.data {
			childKeys.Add(keySegment(key, 0))
		}
	} else {
		for key := range ss.data {
			if len(key) > len(parentPath) && strings.HasPrefix(key, parentPath) && key[len(parentPath)] == ':' {
				childKeys.Add(keySegment(key, len(parentPath)+1))
			}
		}
	}
	container.Sort(childKeys)
	return childKeys
}

func keySegment(key string, prefixLength int) string {
	if prefixLength >= len(key) {
		return ""
	}
	idx := strings.IndexByte(key[prefixLength:], ':')
	if idx == -1 {
		return key[prefixLength:]
	}

	return key[prefixLength : prefixLength+idx]
}
