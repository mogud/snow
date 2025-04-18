package configuration

import (
	"snow/core/container"
	"snow/core/notifier"
	"strings"
	"sync"
)

var _ IConfigurationProvider = (*Provider)(nil)

type Provider struct {
	lock     sync.Mutex
	data     *container.CaseInsensitiveStringMap[string]
	notifier *Notifier
}

func NewProvider() *Provider {
	return &Provider{
		data:     container.NewCaseInsensitiveStringMap[string](),
		notifier: NewNotifier(),
	}
}

func (ss *Provider) Get(key string) string {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	return ss.data.Get(key)
}

func (ss *Provider) TryGet(key string) (value string, ok bool) {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	value, ok = ss.data.TryGet(key)
	return
}

func (ss *Provider) Set(key string, value string) {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	ss.data.Add(key, value)
}

func (ss *Provider) GetReloadNotifier() notifier.INotifier {
	return ss.notifier
}

func (ss *Provider) Replace(data *container.CaseInsensitiveStringMap[string]) {
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

	return getSortedSegmentChildKeys(ss.data, parentPath)
}

func getSortedSegmentChildKeys(m *container.CaseInsensitiveStringMap[string], parentPath string) container.List[string] {
	childKeys := container.NewList[string]()
	if len(parentPath) == 0 {
		m.Scan(func(key, _ string) {
			childKeys.Add(keySegment(key, 0))
		})
	} else {
		upperParentPath := strings.ToUpper(parentPath)
		m.ScanFull(func(upperKey, key, _ string) {
			if len(upperKey) > len(parentPath) && strings.HasPrefix(upperKey, upperParentPath) && upperKey[len(parentPath)] == ':' {
				childKeys.Add(keySegment(key, len(parentPath)+1))
			}
		})
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
