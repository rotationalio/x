package backoff

import (
	"sync"
	"time"
)

// Ticker holds a channel that delivers ticks of a clock at times reported by a backoff
// policy. Ticks will continue to arrive when the previous operation is still running,
// so operations that take a while to fail could run in quick succession.
type Ticker struct {
	C        <-chan time.Time
	b        BackOff
	timer    *time.Timer
	stop     chan struct{}
	stopOnce sync.Once
}

// Returns a new Ticker containing a channel that will send the time at times specified
// by the BackOff policy supplied. Ticker is guaranteed to tick at least once. The
// channel is closed when Stop is called or BackOff stops. It is not safe to manipulate
// the provided backoff policy (e.g. calling NextBackOff or Reset) while ticker is running.
func NewTicker(b BackOff) *Ticker {
	c := make(chan time.Time)
	t := &Ticker{
		C:    c,
		b:    b,
		stop: make(chan struct{}),
	}
	t.b.Reset()
	go t.run(c)
	return t
}

func (t *Ticker) Stop() {
	t.stopOnce.Do(func() { close(t.stop) })
}

func (t *Ticker) run(c chan<- time.Time) {
	defer close(c)

	// Ticker is guaranteed to tick at least once.
	afterC := t.send(c, time.Now())
	for {
		if afterC == nil {
			return
		}

		select {
		case tick := <-afterC:
			afterC = t.send(c, tick)
		case <-t.stop:
			return
		}
	}
}

func (t *Ticker) send(c chan<- time.Time, tick time.Time) <-chan time.Time {
	select {
	case c <- tick:
	case <-t.stop:
		return nil
	}

	delay := t.b.NextBackOff()
	if delay == Stop {
		t.Stop()
		return nil
	}

	// After Go 1.23 there is no reason to prefer a timer over after since the garbage
	// collector can reclaim unreferenced timers.
	if t.timer == nil {
		t.timer = time.NewTimer(delay)
		return t.timer.C
	}

	t.timer.Reset(delay)
	return t.timer.C
}
