package configuration

import (
	"github.com/mogud/snow/core/container"
	"github.com/mogud/snow/core/notifier"
)

type IConfigurationSource interface {
	BuildConfigurationProvider(builder IConfigurationBuilder) IConfigurationProvider
}

type IConfigurationProvider interface {
	Get(key string) string
	TryGet(key string) (value string, ok bool)
	Set(key string, value string)
	GetReloadNotifier() notifier.INotifier
	Load()
	GetChildKeys(parentPath string) container.List[string]
}

type IConfigurationBuilder interface {
	GetProperties() container.Map[string, any]
	GetSources() container.List[IConfigurationSource]
	AddSource(source IConfigurationSource)
	BuildConfigurationRoot() IConfigurationRoot
}

type IConfiguration interface {
	Get(key string) string
	TryGet(key string) (value string, ok bool)
	Set(key string, value string)
	GetSection(key string) IConfigurationSection
	GetChildren() container.List[IConfigurationSection]
	GetChildrenByPath(path string) container.List[IConfigurationSection]
	GetReloadNotifier() notifier.INotifier
}

type IConfigurationRoot interface {
	IConfiguration

	Reload()
	GetProviders() container.List[IConfigurationProvider]
}

type IConfigurationSection interface {
	IConfiguration

	GetKey() string
	GetPath() string
	GetValue() (string, bool)
	SetValue(key string, value string)
}

type IConfigurationManager interface {
	IConfigurationBuilder
	IConfigurationRoot
}
