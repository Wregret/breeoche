package raft

import (
	"sync"
	"time"
)

// FakeClock provides a manually-advanced clock for deterministic tests.
type FakeClock struct {
	mu      sync.Mutex
	now     time.Time
	timers  []*fakeTimer
	tickers []*fakeTicker
}

// NewFakeClock creates a FakeClock starting at the provided time.
func NewFakeClock(start time.Time) *FakeClock {
	return &FakeClock{now: start}
}

func (c *FakeClock) NewTimer(d time.Duration) Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &fakeTimer{
		clock: c,
		when:  c.now.Add(d),
		ch:    make(chan time.Time, 1),
	}
	c.timers = append(c.timers, t)
	return t
}

func (c *FakeClock) NewTicker(d time.Duration) Ticker {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &fakeTicker{
		clock:    c,
		interval: d,
		next:     c.now.Add(d),
		ch:       make(chan time.Time, 1),
	}
	c.tickers = append(c.tickers, t)
	return t
}

// Advance moves the clock forward and fires due timers/tickers.
func (c *FakeClock) Advance(d time.Duration) {
	if d <= 0 {
		return
	}

	c.mu.Lock()
	c.now = c.now.Add(d)
	now := c.now

	dueTimers := []*fakeTimer{}
	activeTimers := c.timers[:0]
	for _, t := range c.timers {
		if t.stopped || t.fired {
			continue
		}
		if !now.Before(t.when) {
			t.fired = true
			dueTimers = append(dueTimers, t)
			continue
		}
		activeTimers = append(activeTimers, t)
	}
	c.timers = activeTimers

	tickerEvents := make(map[*fakeTicker][]time.Time)
	activeTickers := c.tickers[:0]
	for _, t := range c.tickers {
		if t.stopped {
			continue
		}
		for !now.Before(t.next) {
			tickerEvents[t] = append(tickerEvents[t], t.next)
			t.next = t.next.Add(t.interval)
		}
		activeTickers = append(activeTickers, t)
	}
	c.tickers = activeTickers
	c.mu.Unlock()

	for _, t := range dueTimers {
		select {
		case t.ch <- now:
		default:
		}
	}
	for ticker, events := range tickerEvents {
		for _, tick := range events {
			select {
			case ticker.ch <- tick:
			default:
			}
		}
	}
}

type fakeTimer struct {
	clock   *FakeClock
	when    time.Time
	ch      chan time.Time
	stopped bool
	fired   bool
}

func (t *fakeTimer) C() <-chan time.Time {
	return t.ch
}

func (t *fakeTimer) Stop() bool {
	c := t.clock
	c.mu.Lock()
	defer c.mu.Unlock()
	wasActive := !t.stopped && !t.fired
	t.stopped = true
	return wasActive
}

type fakeTicker struct {
	clock    *FakeClock
	interval time.Duration
	next     time.Time
	ch       chan time.Time
	stopped  bool
}

func (t *fakeTicker) C() <-chan time.Time {
	return t.ch
}

func (t *fakeTicker) Stop() {
	c := t.clock
	c.mu.Lock()
	defer c.mu.Unlock()
	t.stopped = true
}
