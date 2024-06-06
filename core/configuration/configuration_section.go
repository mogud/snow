package configuration

import (
	"github.com/mogud/snow/core/container"
	"github.com/mogud/snow/core/notifier"
)

var _ IConfigurationSection = (*Section)(nil)

type Section struct {
	root IConfigurationRoot
	path string
	key  string
}

func NewSection(root IConfigurationRoot, path string) *Section {
	return &Section{
		root: root,
		path: path,
		key:  pathSectionKey(path),
	}
}

func (ss *Section) Get(key string) string {
	return ss.root.Get(ss.path + KeyDelimiter + key)
}

func (ss *Section) TryGet(key string) (value string, ok bool) {
	return ss.root.TryGet(ss.path + KeyDelimiter + key)
}

func (ss *Section) Set(key string, value string) {
	ss.root.Set(ss.path+KeyDelimiter+key, value)
}

func (ss *Section) GetSection(key string) IConfigurationSection {
	return ss.root.GetSection(ss.path + KeyDelimiter + key)
}

func (ss *Section) GetChildren() container.List[IConfigurationSection] {
	return ss.root.GetChildrenByPath(ss.path)
}

func (ss *Section) GetChildrenByPath(path string) container.List[IConfigurationSection] {
	return ss.root.GetChildrenByPath(ss.path + KeyDelimiter + path)
}

func (ss *Section) GetReloadNotifier() notifier.INotifier {
	return ss.root.GetReloadNotifier()
}

func (ss *Section) GetKey() string {
	return ss.key
}

func (ss *Section) GetPath() string {
	return ss.path
}

func (ss *Section) GetValue() (string, bool) {
	return ss.root.TryGet(ss.path)
}

func (ss *Section) SetValue(key, value string) {
	ss.root.Set(key, value)
}
