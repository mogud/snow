package option

type IOptionInjector interface {
	optionInjectorTag()
}

var _ IOptionInjector = (*Option[int])(nil)

type Option[T any] struct {
	factory func(key string) any
	_       T
}

func (ss *Option[T]) Get() T {
	return ss.GetKeyed("")
}

func (ss *Option[T]) GetKeyed(key string) T {
	return ss.factory(key).(T)
}

func (ss *Option[T]) optionInjectorTag() {
}
