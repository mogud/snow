package host

import (
	"gitee.com/mogud/snow/injection"
	"gitee.com/mogud/snow/logging"
	"gitee.com/mogud/snow/logging/handler"
	"gitee.com/mogud/snow/option"
	"reflect"
)

type IBuilder interface {
	GetRoutineCollection() injection.IRoutineCollection
	GetRoutineProvider() injection.IRoutineProvider
	Build() IHost
}

func AddOption[T any](builder IBuilder, path string) {
	repo := injection.GetRoutine[*option.Repository](builder.GetRoutineProvider())
	option.BindOptionPath[T](repo, path)
}

func AddKeyedOption[T any](builder IBuilder, key string, path string) {
	repo := injection.GetRoutine[*option.Repository](builder.GetRoutineProvider())
	option.BindKeyedOptionPath[T](repo, key, path)
}

func AddOptionFactory[T any](builder IBuilder, factory func() T) {
	repo := injection.GetRoutine[*option.Repository](builder.GetRoutineProvider())
	option.BindOptionValue[T](repo, factory())
}

func AddKeyedOptionFactory[T any](builder IBuilder, key string, factory func() T) {
	repo := injection.GetRoutine[*option.Repository](builder.GetRoutineProvider())
	option.BindKeyedOptionValue[T](repo, key, factory())
}

func AddLogHandler[T logging.ILogHandler](builder IBuilder, factory func() T) {
	ch := injection.GetRoutine[*handler.CompoundHandler](builder.GetRoutineProvider())
	ch.AddHandler(factory())
}

func AddLogFormatter(builder IBuilder, name string, formatter func(logData *logging.LogData) string) {
	fr := injection.GetRoutine[*logging.LogFormatterContainer](builder.GetRoutineProvider())
	fr.AddFormatter(name, formatter)
}

func AddSingleton[U any](builder IBuilder) *injection.RoutineDescriptor {
	return addKeyed[U, U](builder, nil, nil, injection.Singleton)
}
func AddVariantSingleton[T, U any](builder IBuilder) *injection.RoutineDescriptor {
	return addKeyed[T, U](builder, nil, nil, injection.Singleton)
}
func AddKeyedSingleton[U any](builder IBuilder, key any) *injection.RoutineDescriptor {
	return addKeyed[U, U](builder, key, nil, injection.Singleton)
}
func AddVariantKeyedSingleton[T, U any](builder IBuilder, key any) *injection.RoutineDescriptor {
	return addKeyed[T, U](builder, key, nil, injection.Singleton)
}
func AddSingletonFactory[U any](builder IBuilder, factory func(scope injection.IRoutineScope) U) *injection.RoutineDescriptor {
	return addKeyed[U, U](builder, nil, factory, injection.Singleton)
}
func AddVariantSingletonFactory[T, U any](builder IBuilder, factory func(scope injection.IRoutineScope) U) *injection.RoutineDescriptor {
	return addKeyed[T, U](builder, nil, factory, injection.Singleton)
}
func AddKeyedSingletonFactory[U any](builder IBuilder, key any, factory func(scope injection.IRoutineScope) U) *injection.RoutineDescriptor {
	return addKeyed[U, U](builder, key, factory, injection.Singleton)
}
func AddVariantKeyedSingletonFactory[T, U any](builder IBuilder, key any, factory func(scope injection.IRoutineScope) U) *injection.RoutineDescriptor {
	return addKeyed[T, U](builder, key, factory, injection.Singleton)
}

func AddScoped[U any](builder IBuilder) *injection.RoutineDescriptor {
	return addKeyed[U, U](builder, nil, nil, injection.Scoped)
}
func AddVariantScoped[T, U any](builder IBuilder) *injection.RoutineDescriptor {
	return addKeyed[T, U](builder, nil, nil, injection.Scoped)
}
func AddKeyedScoped[U any](builder IBuilder, key any) *injection.RoutineDescriptor {
	return addKeyed[U, U](builder, key, nil, injection.Scoped)
}
func AddVariantKeyedScoped[T, U any](builder IBuilder, key any) *injection.RoutineDescriptor {
	return addKeyed[T, U](builder, key, nil, injection.Scoped)
}
func AddScopedFactory[U any](builder IBuilder, factory func(scope injection.IRoutineScope) U) *injection.RoutineDescriptor {
	return addKeyed[U, U](builder, nil, factory, injection.Scoped)
}
func AddVariantScopedFactory[T, U any](builder IBuilder, factory func(scope injection.IRoutineScope) U) *injection.RoutineDescriptor {
	return addKeyed[T, U](builder, nil, factory, injection.Scoped)
}
func AddKeyedScopedFactory[U any](builder IBuilder, key any, factory func(scope injection.IRoutineScope) U) *injection.RoutineDescriptor {
	return addKeyed[U, U](builder, key, factory, injection.Scoped)
}
func AddVariantKeyedScopedFactory[T, U any](builder IBuilder, key any, factory func(scope injection.IRoutineScope) U) *injection.RoutineDescriptor {
	return addKeyed[T, U](builder, key, factory, injection.Scoped)
}

func AddTransient[U any](builder IBuilder) *injection.RoutineDescriptor {
	return addKeyed[U, U](builder, nil, nil, injection.Transient)
}
func AddVariantTransient[T, U any](builder IBuilder) *injection.RoutineDescriptor {
	return addKeyed[T, U](builder, nil, nil, injection.Transient)
}
func AddKeyedTransient[U any](builder IBuilder, key any) *injection.RoutineDescriptor {
	return addKeyed[U, U](builder, key, nil, injection.Transient)
}
func AddVariantKeyedTransient[T, U any](builder IBuilder, key any) *injection.RoutineDescriptor {
	return addKeyed[T, U](builder, key, nil, injection.Transient)
}
func AddTransientFactory[U any](builder IBuilder, factory func(scope injection.IRoutineScope) U) *injection.RoutineDescriptor {
	return addKeyed[U, U](builder, nil, factory, injection.Transient)
}
func AddVariantTransientFactory[T, U any](builder IBuilder, factory func(scope injection.IRoutineScope) U) *injection.RoutineDescriptor {
	return addKeyed[T, U](builder, nil, factory, injection.Transient)
}
func AddKeyedTransientFactory[U any](builder IBuilder, key any, factory func(scope injection.IRoutineScope) U) *injection.RoutineDescriptor {
	return addKeyed[U, U](builder, key, factory, injection.Transient)
}
func AddVariantKeyedTransientFactory[T, U any](builder IBuilder, key any, factory func(scope injection.IRoutineScope) U) *injection.RoutineDescriptor {
	return addKeyed[T, U](builder, key, factory, injection.Transient)
}

func addKeyed[T, U any](
	builder IBuilder, key any, factory func(scope injection.IRoutineScope) U, lifetime injection.RoutineLifetime,
) *injection.RoutineDescriptor {
	tyKey := reflect.TypeOf((*T)(nil)).Elem()
	tyImpl := reflect.TypeOf((*U)(nil)).Elem()

	if key == nil {
		key = injection.DefaultKey
	}
	var untypedFactory func(scope injection.IRoutineScope) any
	if factory == nil {
		untypedFactory = func(scope injection.IRoutineScope) any {
			return reflect.New(tyImpl.Elem()).Interface()
		}
	} else {
		untypedFactory = func(scope injection.IRoutineScope) any {
			return factory(scope)
		}
	}

	desc := &injection.RoutineDescriptor{
		Lifetime: lifetime,
		Key:      key,
		TyKey:    tyKey,
		TyImpl:   tyImpl,
		Factory:  untypedFactory,
	}
	builder.GetRoutineCollection().AddDescriptor(desc)
	return desc
}
