package sources

import (
	"snow/core/configuration"
)

var _ configuration.IConfigurationSource = (*MemoryConfigurationSource)(nil)

type MemoryConfigurationSource struct {
	InitData map[string]string
}

func (ss *MemoryConfigurationSource) BuildConfigurationProvider(builder configuration.IConfigurationBuilder) configuration.IConfigurationProvider {
	return NewMemoryConfigurationProvider(ss)
}

var _ configuration.IConfigurationProvider = (*MemoryConfigurationProvider)(nil)

type MemoryConfigurationProvider struct {
	*configuration.Provider
}

func NewMemoryConfigurationProvider(source *MemoryConfigurationSource) *MemoryConfigurationProvider {
	provider := configuration.NewProvider()
	if source.InitData != nil {
		for k, v := range source.InitData {
			provider.Set(k, v)
		}
	}
	return &MemoryConfigurationProvider{
		Provider: provider,
	}
}
