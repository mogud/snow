package configuration

import (
	"github.com/mogud/snow/core/container"
	"github.com/mogud/snow/core/notifier"
)

var _ notifier.INotifier = (*Notifier)(nil)

type Notifier struct {
	callbacks container.List[func()]
}

func NewNotifier() *Notifier {
	return &Notifier{}
}

func (ss *Notifier) RegisterNotifyCallback(callback func()) {
	ss.callbacks = append(ss.callbacks, callback)
}

func (ss *Notifier) Notify() {
	for _, callback := range ss.callbacks {
		callback()
	}
}
