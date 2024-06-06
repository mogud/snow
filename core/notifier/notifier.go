package notifier

type INotifier interface {
	RegisterNotifyCallback(callback func())
}
