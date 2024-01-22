package builder

import (
	"snow/core/logging"
	"snow/core/logging/handler"
	"snow/core/logging/handler/console"
	"snow/core/logging/handler/file"
	"snow/core/logging/slog"
	"snow/core/option"
	"snow/host"
	"snow/host/internal"
	"snow/injection"
)

var _ host.IBuilder = (*DefaultBuilder)(nil)

type DefaultBuilder struct {
	descriptors injection.IRoutineCollection

	provider injection.IRoutineProvider
}

func NewDefaultBuilder() *DefaultBuilder {
	builder := &DefaultBuilder{
		descriptors: internal.NewRoutineCollection(),
	}
	builder.provider = internal.NewProvider(builder.descriptors, nil)

	host.AddSingletonFactory[*option.Repository](builder, func(scope injection.IRoutineScope) *option.Repository {
		return option.NewOptionRepository()
	})

	host.AddSingletonFactory[*logging.LogFormatterContainer](builder, func(scope injection.IRoutineScope) *logging.LogFormatterContainer {
		f := logging.NewLogFormatterRepository()
		f.AddFormatter("Default", logging.DefaultLogFormatter)
		f.AddFormatter("Color", logging.ColorLogFormatter)
		return f
	})

	host.AddOption[*console.Option](builder, "Log.Console")
	host.AddSingletonFactory[*console.Handler](builder, func(scope injection.IRoutineScope) *console.Handler {
		return console.NewHandler()
	})
	host.AddOption[*console.Option](builder, "Log.File")
	host.AddSingletonFactory[*file.Handler](builder, func(scope injection.IRoutineScope) *file.Handler {
		return file.NewHandler()
	})
	host.AddSingletonFactory[*handler.CompoundHandler](builder, func(scope injection.IRoutineScope) *handler.CompoundHandler {
		ch := injection.GetRoutine[*console.Handler](builder.provider)
		fh := injection.GetRoutine[*file.Handler](builder.provider)

		compoundHandler := handler.NewCompoundHandler()
		compoundHandler.AddHandler(ch)
		compoundHandler.AddHandler(fh)

		slog.BindGlobalHandler(compoundHandler)

		return compoundHandler
	})

	host.AddVariantSingleton[host.IHostedRoutineContainer, *internal.HostedRoutineContainer](builder)
	host.AddVariantSingleton[host.IHostedLifecycleRoutineContainer, *internal.HostedLifecycleRoutineContainer](builder)

	return builder
}

func (ss *DefaultBuilder) GetRoutineProvider() injection.IRoutineProvider {
	return ss.provider
}

func (ss *DefaultBuilder) GetRoutineCollection() injection.IRoutineCollection {
	return ss.descriptors
}

func (ss *DefaultBuilder) Build() host.IHost {
	host.AddOption[*internal.HostOption](ss, "Host")

	host.AddSingletonFactory[host.IHost](ss, func(scope injection.IRoutineScope) host.IHost { return internal.NewHost(ss.provider) })
	host.AddSingletonFactory[host.IHostApplication](ss, func(scope injection.IRoutineScope) host.IHostApplication {
		return internal.NewHostApplication()
	})
	host.AddHostedRoutine[*internal.ConsoleLifetimeRoutine](ss)

	return host.GetRoutine[host.IHost](ss.provider)
}
