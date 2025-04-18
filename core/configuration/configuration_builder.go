package configuration

import "snow/core/container"

var _ IConfigurationBuilder = (*Builder)(nil)

type Builder struct {
	properties container.Map[string, any]
	sources    container.List[IConfigurationSource]
}

func NewBuilder() *Builder {
	return &Builder{
		properties: container.NewMap[string, any](),
		sources:    container.NewList[IConfigurationSource](),
	}
}

func (ss *Builder) GetProperties() container.Map[string, any] {
	return ss.properties
}

func (ss *Builder) GetSources() container.List[IConfigurationSource] {
	return ss.sources
}

func (ss *Builder) AddSource(source IConfigurationSource) {
	ss.sources.Add(source)
}

func (ss *Builder) BuildConfigurationRoot() IConfigurationRoot {
	providers := container.NewList[IConfigurationProvider]()
	for _, source := range ss.sources {
		providers.Add(source.BuildConfigurationProvider(ss))
	}
	return NewConfigurationRoot(providers)
}
