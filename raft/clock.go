package raft

import "time"

// Clock abstracts time for deterministic tests.
type Clock interface {
	NewTimer(d time.Duration) Timer
	NewTicker(d time.Duration) Ticker
}

// Timer abstracts time.Timer for testing.
type Timer interface {
	C() <-chan time.Time
	Stop() bool
}

// Ticker abstracts time.Ticker for testing.
type Ticker interface {
	C() <-chan time.Time
	Stop()
}

type realClock struct{}

func (realClock) NewTimer(d time.Duration) Timer {
	return realTimer{t: time.NewTimer(d)}
}

func (realClock) NewTicker(d time.Duration) Ticker {
	return realTicker{t: time.NewTicker(d)}
}

type realTimer struct {
	t *time.Timer
}

func (t realTimer) C() <-chan time.Time {
	return t.t.C
}

func (t realTimer) Stop() bool {
	return t.t.Stop()
}

type realTicker struct {
	t *time.Ticker
}

func (t realTicker) C() <-chan time.Time {
	return t.t.C
}

func (t realTicker) Stop() {
	t.t.Stop()
}
