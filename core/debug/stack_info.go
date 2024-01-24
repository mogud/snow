package debug

import (
	"runtime"
)

func StackInfo() string {
	buf := make([]byte, 4096)
	buf = buf[:runtime.Stack(buf, false)]
	return string(buf)
}
