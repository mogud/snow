package sources

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/mogud/snow/core/configuration"
	"github.com/mogud/snow/core/container"
	stripjsoncomments "github.com/trapcodeio/go-strip-json-comments"
	"log"
)

var _ configuration.IConfigurationSource = (*JsonConfigurationSource)(nil)

type JsonConfigurationSource struct {
	Path           string
	Optional       bool
	ReloadOnChange bool
}

func (ss *JsonConfigurationSource) BuildConfigurationProvider(_ configuration.IConfigurationBuilder) configuration.IConfigurationProvider {
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

	newMap, err := ConvertJsonToConfigurationKV("", jsonWithoutComments)
	if err != nil {
		log.Printf("load json: %v", err)
		ss.Replace(container.NewCaseInsensitiveStringMap[string]())
		return
	}

	ss.Replace(newMap)
}

func ConvertJsonToConfigurationKV(head string, json string) (*container.CaseInsensitiveStringMap[string], error) {
	var jsons map[string]interface{}
	if err := jsoniter.UnmarshalFromString(json, &jsons); err != nil {
		return nil, fmt.Errorf("json unmashal failed: %v\n", err)
	}

	newMap := container.NewCaseInsensitiveStringMap[string]()
	for key, value := range jsons {
		if len(head) == 0 {
			fillMap(newMap, key, value)
		} else {
			fillMap(newMap, fmt.Sprintf("%s:%s", head, key), value)
		}
	}
	return newMap, nil
}

func fillMap(m *container.CaseInsensitiveStringMap[string], key string, value interface{}) {
	switch v := value.(type) {
	case string:
		m.Add(key, v)
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
			m.Add(key, fmt.Sprintf("%d", n))
			return
		}

		m.Add(key, fmt.Sprintf("%f", v))
	case bool:
		m.Add(key, fmt.Sprintf("%t", v))
	default:
		panic(fmt.Errorf("invalid type: %T => %v", v, v))
	}
}
