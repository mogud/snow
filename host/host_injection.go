package host

import (
	"gitee.com/mogud/snow/core/logging"
	"gitee.com/mogud/snow/core/logging/handler"
	"gitee.com/mogud/snow/core/option"
	"gitee.com/mogud/snow/injection"
	"reflect"
	"strings"
)

var optionContainerType = reflect.TypeOf((*option.IOptionInjector)(nil)).Elem()
var loggerContainerType = reflect.TypeOf((*logging.ILoggerInjector)(nil)).Elem()

func Inject(scope injection.IRoutineScope, instance any) bool {
	v := reflect.ValueOf(instance)
	vTy := v.Type()

	for i := 0; i < vTy.NumMethod(); i++ {
		fMethod := vTy.Method(i)
		if strings.HasPrefix(fMethod.Name, "Construct") {
			fTy := fMethod.Type
			args := make([]reflect.Value, 0, fTy.NumIn())
			args = append(args, v)
			for j := 1; j < fTy.NumIn(); j++ {
				argTy := fTy.In(j)
				var argInstance any
				switch {
				case argTy.ConvertibleTo(optionContainerType):
					repo := injection.GetRoutine[*option.Repository](scope.GetRoot().GetProvider())
					argInstance = repo.GetOption(argTy)
				case argTy.ConvertibleTo(loggerContainerType):
					ch := injection.GetRoutine[*handler.CompoundHandler](scope.GetRoot().GetProvider())
					argInstance = ch.WrapToContainer(argTy)
				default:
					argInstance = scope.GetProvider().GetRoutine(argTy)
				}

				if argInstance == nil {
					args = append(args, reflect.Zero(argTy))
				} else {
					args = append(args, reflect.ValueOf(argInstance))
				}
			}
			fMethod.Func.Call(args)
		}
	}
	return false
}

// NewStruct 通过反射创建指定类型 T 的实例，类型 T 必须为结构体指针
func NewStruct[T any]() T {
	ty := reflect.TypeOf((*T)(nil)).Elem()
	return reflect.New(ty.Elem()).Interface().(T)
}
