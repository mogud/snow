package option

import (
	"fmt"
	jsonparser "github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
	stripjsoncomments "github.com/trapcodeio/go-strip-json-comments"
	"os"
	"reflect"
)

var k = koanf.New(".")

var typeKeyBinding = make(map[reflect.Type]string)

// GetByKey 通过 key 返回配置
func GetByKey(key string, inout interface{}) error {
	raw := k.Get(key)
	_ = raw

	if err := k.Unmarshal(key, inout); err != nil {
		fmt.Printf("get key: %+v", inout)
		return err
	}
	return nil
}

// Get 通过类型获取配置
func Get[T any]() (out *T, err error) {
	optionType := reflect.TypeOf((*T)(nil))
	out = (*T)(reflect.New(optionType.Elem()).UnsafePointer())
	if key, ok := typeKeyBinding[optionType]; ok {
		err = GetByKey(key, out)
	}

	return
}

// GetInout 通过传入 inout 变量类型获取配置，并写入 inout
func GetInout(inout any) error {
	optionType := reflect.TypeOf(inout)
	if key, ok := typeKeyBinding[optionType]; ok {
		return GetByKey(key, inout)
	}
	return nil
}

// BindType 将类型绑定到 key，而后通过 Get 可以直接获取配置数据
func BindType[T any](key string) {
	typeKeyBinding[reflect.TypeOf((*T)(nil))] = key
}

func AddFile(filePath string, parser koanf.Parser) error {
	rawBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file(%v): %w", filePath, err)
	}

	err = k.Load(rawbytes.Provider(rawBytes), parser)
	if err != nil {
		return fmt.Errorf("failed to parse file(%s): %w", filePath, err)
	}
	return nil
}

func AddJSONFile(filePath string) error {
	jsonRawBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read json file(%v): %w", filePath, err)
	}

	jsonWithoutComments := stripjsoncomments.Strip(string(jsonRawBytes))
	err = k.Load(rawbytes.Provider([]byte(jsonWithoutComments)), jsonparser.Parser())
	if err != nil {
		return fmt.Errorf("failed to parse json file(%s): %w", filePath, err)
	}
	return nil
}
