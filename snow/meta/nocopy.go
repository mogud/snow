package meta

import "sync"

var _ = sync.Locker((*NoCopy)(nil))

type NoCopy struct{}

func (*NoCopy) Lock()   {}
func (*NoCopy) Unlock() {}
