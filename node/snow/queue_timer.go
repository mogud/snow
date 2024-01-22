package snow

type timerItem struct {
	idx        int64
	timeMs     int64
	intervalMs int64
	start      bool
	fun        func()
}

type timerQueue []*timerItem

func (ss timerQueue) Len() int {
	return len(ss)
}

func (ss timerQueue) Less(i, j int) bool {
	l, r := ss[i], ss[j]
	return l.timeMs < r.timeMs || l.timeMs == r.timeMs && l.idx < r.idx
}

func (ss timerQueue) Swap(i, j int) {
	ss[i], ss[j] = ss[j], ss[i]
}

func (ss *timerQueue) Push(x interface{}) {
	item := x.(*timerItem)
	*ss = append(*ss, item)
}

func (ss *timerQueue) Pop() interface{} {
	old := *ss
	n := len(old)
	x := old[n-1]
	*ss = old[:n-1]
	return x
}
