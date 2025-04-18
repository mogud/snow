package configuration

import (
	"reflect"
	"strconv"
	"strings"
	"time"
)

func GetBool(config IConfiguration, key string) bool {
	v, _ := TryGetBool(config, key)
	return v
}

func TryGetBool(config IConfiguration, key string) (bool, bool) {
	v, ok := config.TryGet(key)
	if !ok {
		return false, false
	}
	return strings.ToUpper(v) == "TRUE", true
}

func GetInt64(config IConfiguration, key string) int64 {
	nv, _ := TryGetInt64(config, key)
	return nv
}

func TryGetInt64(config IConfiguration, key string) (int64, bool) {
	v, ok := config.TryGet(key)
	if !ok {
		return 0, false
	}
	nv, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, false
	}
	return nv, true
}

func GetUint64(config IConfiguration, key string) uint64 {
	nv, _ := TryGetUint64(config, key)
	return nv
}

func TryGetUint64(config IConfiguration, key string) (uint64, bool) {
	v, ok := config.TryGet(key)
	if !ok {
		return 0, false
	}
	nv, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0, false
	}
	return nv, true
}

func GetFloat64(config IConfiguration, key string) float64 {
	nv, _ := TryGetFloat64(config, key)
	return nv
}

func TryGetFloat64(config IConfiguration, key string) (float64, bool) {
	v, ok := config.TryGet(key)
	if !ok {
		return 0, false
	}
	nv, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, false
	}
	return nv, true
}

func Get[T any](root IConfiguration, path string) T {
	ty := reflect.TypeOf((*T)(nil)).Elem()
	return GetByType(ty, root, path).(T)
}

func GetByType(ty reflect.Type, root IConfiguration, path string) any {
	val := reflect.New(ty).Elem()
	fillValue(val, root, path)
	return val.Interface()
}

func Fill(root IConfiguration, path string, out any) {
	val := reflect.ValueOf(out).Elem()
	fillValue(val, root, path)
}

func fillValue(val reflect.Value, config IConfiguration, key string) {
	ty := val.Type()
	switch ty.Kind() {
	case reflect.String:
		if v, ok := config.TryGet(key); ok {
			val.SetString(v)
		}
	case reflect.Bool:
		if v, ok := TryGetBool(config, key); ok {
			val.SetBool(v)
		}
	case reflect.Int:
		if v, ok := TryGetInt64(config, key); ok {
			val.SetInt((int64)((int)(v)))
		}
	case reflect.Int8:
		if v, ok := TryGetInt64(config, key); ok {
			val.SetInt((int64)((int8)(v)))
		}
	case reflect.Int16:
		if v, ok := TryGetInt64(config, key); ok {
			val.SetInt((int64)((int16)(v)))
		}
	case reflect.Int32:
		if v, ok := TryGetInt64(config, key); ok {
			val.SetInt((int64)((int32)(v)))
		}
	case reflect.Int64:
		if v, ok := TryGetInt64(config, key); ok {
			val.SetInt(v)
		}
	case reflect.Uint:
		if v, ok := TryGetUint64(config, key); ok {
			val.SetUint((uint64)((uint)(v)))
		}
	case reflect.Uint8:
		if v, ok := TryGetUint64(config, key); ok {
			val.SetUint((uint64)((uint8)(v)))
		}
	case reflect.Uint16:
		if v, ok := TryGetUint64(config, key); ok {
			val.SetUint((uint64)((uint16)(v)))
		}
	case reflect.Uint32:
		if v, ok := TryGetUint64(config, key); ok {
			val.SetUint((uint64)((uint32)(v)))
		}
	case reflect.Uint64:
		if v, ok := TryGetUint64(config, key); ok {
			val.SetUint(v)
		}
	case reflect.Float32:
		if v, ok := TryGetFloat64(config, key); ok {
			val.SetFloat((float64)((float32)(v)))
		}
	case reflect.Float64:
		if v, ok := TryGetFloat64(config, key); ok {
			val.SetFloat(v)
		}
	case reflect.Map:
		fillMap(ty, val, config, key)
	case reflect.Pointer:
		pv := reflect.New(ty.Elem())
		v := pv.Elem()
		fillValue(v, config, key)
		val.Set(pv)
	case reflect.Slice:
		children := config.GetSection(key).GetChildren()
		slice := reflect.MakeSlice(ty, 0, len(children))
		for i := 0; i < children.Len(); i++ {
			sv := reflect.New(ty.Elem()).Elem()
			fillValue(sv, config.GetSection(key), strconv.Itoa(i))
			slice = reflect.Append(slice, sv)
		}
		val.Set(slice)
	case reflect.Struct:
		fillStruct(ty, val, config, key)
	default:
		panic("unsupported type")
	}
}

func fillMap(ty reflect.Type, val reflect.Value, config IConfiguration, key string) {
	mapSection := config.GetSection(key)
	children := mapSection.GetChildren()
	m := reflect.MakeMap(ty)
	for _, child := range children {
		mKey := child.GetKey()
		k := reflect.New(ty.Key()).Elem()
		switch ty.Key().Kind() {
		case reflect.String:
			k.SetString(mKey)
		case reflect.Int:
			v, _ := strconv.ParseInt(mKey, 10, 64)
			k.SetInt((int64)((int)(v)))
		case reflect.Int8:
			v, _ := strconv.ParseInt(mKey, 10, 64)
			k.SetInt((int64)((int8)(v)))
		case reflect.Int16:
			v, _ := strconv.ParseInt(mKey, 10, 64)
			k.SetInt((int64)((int16)(v)))
		case reflect.Int32:
			v, _ := strconv.ParseInt(mKey, 10, 64)
			k.SetInt((int64)((int32)(v)))
		case reflect.Int64:
			v, _ := strconv.ParseInt(mKey, 10, 64)
			k.SetInt(v)
		case reflect.Uint:
			v, _ := strconv.ParseInt(mKey, 10, 64)
			k.SetInt((int64)((uint)(v)))
		case reflect.Uint8:
			v, _ := strconv.ParseInt(mKey, 10, 64)
			k.SetInt((int64)((uint8)(v)))
		case reflect.Uint16:
			v, _ := strconv.ParseInt(mKey, 10, 64)
			k.SetInt((int64)((uint16)(v)))
		case reflect.Uint32:
			v, _ := strconv.ParseInt(mKey, 10, 64)
			k.SetInt((int64)((uint32)(v)))
		case reflect.Uint64:
			v, _ := strconv.ParseInt(mKey, 10, 64)
			k.SetInt((int64)((uint64)(v)))
		default:
			panic("unhandled default case")
		}

		v := reflect.New(ty.Elem()).Elem()
		fillValue(v, config.GetSection(key), mKey)

		m.SetMapIndex(k, v)
	}
	val.Set(m)
}

func fillStruct(ty reflect.Type, val reflect.Value, config IConfiguration, key string) {
	var vs reflect.Value

	section := config
	if len(key) > 0 {
		section = config.GetSection(key)
	}

	var isTime bool
	if ty.Name() == "Time" && ty.PkgPath() == "time" {
		isTime = true
		if v, ok := config.TryGet(key); ok {
			t, err := time.Parse(time.RFC3339, v)
			if err == nil {
				vs = reflect.ValueOf(t)
			}
		}
	}

	if !vs.IsValid() {
		vs = reflect.New(ty).Elem()
	}

	if !isTime {
		for i := 0; i < ty.NumField(); i++ {
			fv := vs.Field(i)
			ft := ty.Field(i)
			if !ft.IsExported() {
				continue
			}

			fName := ft.Tag.Get("snow")
			if fName == "" {
				fName = ft.Name
			}
			fillValue(fv, section, fName)
		}
	}

	val.Set(vs)
}
