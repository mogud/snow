package option

type IOptionInjector interface {
	optionInjectorTag()
}

var _ IOptionInjector = (*Option[int])(nil)

type Option[T any] struct {
	factory          func(key string) any
	registerOnChange func(key string, cb func())
	_                T
}

func (ss *Option[T]) Get() T {
	return ss.GetKeyed("")
}

func (ss *Option[T]) GetKeyed(key string) T {
	return ss.factory(key).(T)
}

func (ss *Option[T]) OnChanged(cb func()) {
	ss.registerOnChange("", cb)
}

func (ss *Option[T]) OnKeyedChanged(key string, cb func()) {
	ss.registerOnChange(key, cb)
}

func (ss *Option[T]) optionInjectorTag() {
}
