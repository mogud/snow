package sources

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/mogud/snow/core/configuration"
	"github.com/mogud/snow/core/container"
	stripjsoncomments "github.com/trapcodeio/go-strip-json-comments"
	"strings"
)

var _ configuration.IConfigurationSource = (*JsonConfigurationSource)(nil)

type JsonConfigurationSource struct {
	Path           string
	Optional       bool
	ReloadOnChange bool
}

func (ss *JsonConfigurationSource) BuildConfigurationProvider(builder configuration.IConfigurationBuilder) configuration.IConfigurationProvider {
	return NewJsonConfigurationProvider(ss)
}

var _ configuration.IConfigurationProvider = (*JsonConfigurationProvider)(nil)

type JsonConfigurationProvider struct {
	*FileConfigurationProvider
}

func NewJsonConfigurationProvider(source *JsonConfigurationSource) configuration.IConfigurationProvider {
	provider := &JsonConfigurationProvider{
		FileConfigurationProvider: NewFileConfigurationProvider(&FileConfigurationSource{
			Path:           source.Path,
			Optional:       source.Optional,
			ReloadOnChange: source.ReloadOnChange,
		}),
	}
	provider.OnLoad = provider.OnLoadJson
	return provider
}

func (ss *JsonConfigurationProvider) OnLoadJson(bytes []byte) {
	jsonWithoutComments := stripjsoncomments.Strip(string(bytes))
	var jsons map[string]interface{}
	if err := jsoniter.UnmarshalFromString(jsonWithoutComments, &jsons); err != nil {
		fmt.Printf("json unmashal failed: %v\n", err)
		ss.Replace(container.NewMap[string, string]())
		return
	}

	newMap := make(map[string]string)
	for key, value := range jsons {
		fillMap(newMap, key, value)
	}
	ss.Replace(newMap)
}

func fillMap(m map[string]string, key string, value interface{}) {
	switch v := value.(type) {
	case string:
		m[strings.ToUpper(key)] = v
	case map[string]interface{}:
		for k, v := range v {
			fillMap(m, fmt.Sprintf("%s:%s", key, k), v)
		}
	case []interface{}:
		for i, v := range v {
			fillMap(m, fmt.Sprintf("%s:%d", key, i), v)
		}
	case float64:
		n := int64(v)
		if v == float64(n) {
			m[strings.ToUpper(key)] = fmt.Sprintf("%d", n)
			return
		}

		m[strings.ToUpper(key)] = fmt.Sprintf("%f", v)
	case bool:
		m[strings.ToUpper(key)] = fmt.Sprintf("%t", v)
	default:
		panic(fmt.Errorf("invalid type: %T => %v", v, v))
	}
}
