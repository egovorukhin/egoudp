package timer

import (
	"sync"
	"time"
)

type Timer struct {
	Count    int
	Duration int
	Interval time.Duration
	sync.Mutex
	stop bool
}

func NewTimer(d int, interval time.Duration) *Timer {
	return &Timer{
		Duration: d,
		Interval: interval,
		stop:     false,
	}
}

func (t *Timer) Start(f func()) {
	for !t.stop {
		if t.Count == t.Duration {
			f()
			t.Count = 0
		}
		t.Count++
		time.Sleep(t.Interval)
	}
}

func (t *Timer) Reset() {
	t.Count = 0
}

func (t *Timer) Stop() {
	t.Lock()
	t.stop = true
	t.Unlock()
}
