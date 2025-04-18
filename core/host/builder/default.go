package builder

import (
	"snow/core/configuration"
	"snow/core/host"
	"snow/core/host/internal"
	"snow/core/injection"
	"snow/core/logging"
	"snow/core/logging/handler"
	"snow/core/logging/handler/compound"
	"snow/core/logging/handler/console"
	"snow/core/logging/handler/file"
	"snow/core/logging/slog"
	"snow/core/option"
)

var _ host.IBuilder = (*DefaultBuilder)(nil)

type DefaultBuilder struct {
	descriptors injection.IRoutineCollection

	config *configuration.Manager

	provider injection.IRoutineProvider
}

func NewDefaultBuilder() *DefaultBuilder {
	builder := &DefaultBuilder{
		descriptors: internal.NewRoutineCollection(),
		config:      configuration.NewManager(),
	}
	builder.provider = internal.NewProvider(builder.descriptors, nil)

	host.AddVariantSingletonFactory[configuration.IConfiguration](builder, func(scope injection.IRoutineScope) configuration.IConfiguration {
		return builder.config
	})

	host.AddSingletonFactory[*option.Repository](builder, func(scope injection.IRoutineScope) *option.Repository {
		return option.NewOptionRepository(builder.config)
	})

	host.AddSingletonFactory[*logging.LogFormatterContainer](builder, func(scope injection.IRoutineScope) *logging.LogFormatterContainer {
		f := logging.NewLogFormatterRepository()
		f.AddFormatter("Default", logging.DefaultLogFormatter)
		f.AddFormatter("Color", logging.ColorLogFormatter)
		return f
	})

	host.AddOption[*compound.Option](builder, "Log:Compound")
	host.AddSingletonFactory[*compound.Handler](builder, func(scope injection.IRoutineScope) *compound.Handler {
		return compound.NewHandler()
	})
	host.AddOption[*console.Option](builder, "Log:Console")
	host.AddSingletonFactory[*console.Handler](builder, func(scope injection.IRoutineScope) *console.Handler {
		return console.NewHandler()
	})
	host.AddOption[*file.Option](builder, "Log:File")
	host.AddSingletonFactory[*file.Handler](builder, func(scope injection.IRoutineScope) *file.Handler {
		return file.NewHandler()
	})
	host.AddSingletonFactory[*handler.RootHandler](builder, func(scope injection.IRoutineScope) *handler.RootHandler {
		compoundH := injection.GetRoutine[*compound.Handler](builder.provider)
		ch := injection.GetRoutine[*console.Handler](builder.provider)
		fh := injection.GetRoutine[*file.Handler](builder.provider)

		compoundH.AddHandler(ch)
		compoundH.AddHandler(fh)

		compoundHandler := handler.NewRootHandler(compoundH)
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

func (ss *DefaultBuilder) GetConfigurationManager() configuration.IConfigurationManager {
	return ss.config
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
