package compound

import (
	"github.com/mogud/snow/core/container"
	"github.com/mogud/snow/core/logging"
	"github.com/mogud/snow/core/option"
)

var _ logging.ILogHandler = (*Handler)(nil)

type Option struct {
	NodeId   int    `snow:"NodeId"`   // 日志节点 Id
	NodeName string `snow:"NodeName"` // 日志节点名
}

type Handler struct {
	proxy container.List[logging.ILogHandler]
	opt   *Option
}

func NewHandler() *Handler {
	return &Handler{}
}

func (ss *Handler) Construct(opt *option.Option[*Option]) {
	ss.opt = opt.Get()
}

func (ss *Handler) Log(data *logging.LogData) {
	data.NodeID = ss.opt.NodeId
	data.NodeName = ss.opt.NodeName
	for _, handler := range ss.proxy {
		handler.Log(data)
	}
}

func (ss *Handler) AddHandler(logger logging.ILogHandler) {
	ss.proxy.Add(logger)
}
