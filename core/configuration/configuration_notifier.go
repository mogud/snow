package configuration

import (
	"snow/core/container"
	"snow/core/notifier"
	"sync"
)

var _ notifier.INotifier = (*Notifier)(nil)

type Notifier struct {
	lock      sync.Mutex
	callbacks container.List[func()]
}

func NewNotifier() *Notifier {
	return &Notifier{}
}

func (ss *Notifier) RegisterNotifyCallback(callback func()) {
	ss.lock.Lock()
	defer ss.lock.Unlock()
	ss.callbacks = append(ss.callbacks, callback)
}

func (ss *Notifier) Notify() {
	var cbs container.List[func()]
	ss.lock.Lock()
	cbs = ss.callbacks
	ss.lock.Unlock()

	for _, callback := range cbs {
		callback()
	}
}
