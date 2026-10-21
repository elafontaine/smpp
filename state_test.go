package smpp

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestStateStartsWithItsInitialValue(t *testing.T) {
	s := NewESMEState("OPEN")
	if got := s.GetState(); got != "OPEN" {
		t.Errorf("GetState() = %q, want OPEN", got)
	}
}

func TestSetStateIsVisibleToGetState(t *testing.T) {
	s := NewESMEState("OPEN")
	s.SetState("BOUND_TX")
	if got := s.GetState(); got != "BOUND_TX" {
		t.Errorf("GetState() = %q, want BOUND_TX", got)
	}
}

// Each Transition sees the state left by the previous one, so concurrent
// read-modify-write sequences never lose an update (run with -race).
func TestTransitionIsAtomic(t *testing.T) {
	s := NewESMEState("0")
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Transition(func(current string) string {
				n, _ := strconv.Atoi(current)
				return strconv.Itoa(n + 1)
			})
		}()
	}
	wg.Wait()
	if got := s.GetState(); got != "100" {
		t.Errorf("GetState() = %q, want 100", got)
	}
}

// The case that matters for SMPP: of several concurrent binds, only one may
// move the session from OPEN to bound.
func TestOnlyOneConcurrentBindWins(t *testing.T) {
	s := NewESMEState("OPEN")
	const binds = 10
	won := make(chan bool, binds)
	for i := 0; i < binds; i++ {
		go func() {
			previous, _ := s.Transition(func(current string) string {
				if current == "OPEN" {
					return "BOUND_TX"
				}
				return current
			})
			won <- previous == "OPEN"
		}()
	}
	winners := 0
	for i := 0; i < binds; i++ {
		select {
		case w := <-won:
			if w {
				winners++
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out")
		}
	}
	if winners != 1 {
		t.Errorf("%d binds won, want exactly 1", winners)
	}
}

func TestClosedIsTerminal(t *testing.T) {
	s := NewESMEState("BOUND_TX")
	s.SetState(CLOSED)
	s.SetState(CLOSED) // safe to repeat
	s.SetState("OPEN") // too late: CLOSED is terminal
	if got := s.GetState(); got != CLOSED {
		t.Errorf("GetState() = %q, want %q", got, CLOSED)
	}
	called := false
	s.Transition(func(string) string { called = true; return "OPEN" })
	if called {
		t.Error("the callback ran on a CLOSED state")
	}
}

func TestStateIsHandedBackWhenTheCallbackPanics(t *testing.T) {
	s := NewESMEState("OPEN")
	func() {
		defer func() { recover() }()
		s.Transition(func(string) string { panic("boom") })
	}()

	got := make(chan string, 1)
	go func() { got <- s.GetState() }()
	select {
	case state := <-got:
		if state != "OPEN" {
			t.Errorf("GetState() = %q, want OPEN", state)
		}
	case <-time.After(time.Second):
		t.Fatal("the state was lost when the callback panicked")
	}
}
