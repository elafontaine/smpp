package smpp

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestStateStartsWithItsInitialValue(t *testing.T) {
	s := newSessionState("OPEN")
	defer s.SetState(CLOSED) // stops the state\'s goroutine
	if got := s.GetState(); got != "OPEN" {
		t.Errorf("GetState() = %q, want OPEN", got)
	}
}

func TestSetStateIsVisibleToGetState(t *testing.T) {
	s := newSessionState("OPEN")
	defer s.SetState(CLOSED) // stops the state\'s goroutine
	s.SetState("BOUND_TX")
	if got := s.GetState(); got != "BOUND_TX" {
		t.Errorf("GetState() = %q, want BOUND_TX", got)
	}
}

// Each Transition sees the state left by the previous one, so concurrent
// read-modify-write sequences never lose an update (run with -race).
func TestTransitionIsAtomic(t *testing.T) {
	s := newSessionState("0")
	defer s.SetState(CLOSED) // stops the state\'s goroutine
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
	s := newSessionState("OPEN")
	defer s.SetState(CLOSED) // stops the state\'s goroutine
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
	s := newSessionState("BOUND_TX")
	defer s.SetState(CLOSED) // stops the state\'s goroutine
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
	s := newSessionState("OPEN")
	defer s.SetState(CLOSED) // stops the state\'s goroutine
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

	// Reads would still work if the panicking transition never finished; the next transition
	// would not.
	done := make(chan struct{})
	go func() { s.SetState("BOUND_TX"); close(done) }()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("transitions are stuck after the callback panicked")
	}
}

// Done lets a select wait for CLOSED instead of polling GetState.
func TestDoneIsClosedWhenTheStateBecomesClosed(t *testing.T) {
	s := newSessionState(OPEN)
	defer s.SetState(CLOSED) // stops the state\'s goroutine
	s.SetState(BOUND_TX)
	select {
	case <-s.Done():
		t.Fatal("Done() closed before the state became CLOSED")
	default:
	}

	s.SetState(CLOSED)
	select {
	case <-s.Done():
	case <-time.After(time.Second):
		t.Fatal("Done() not closed after the state became CLOSED")
	}
	s.SetState(CLOSED) // again: must not close Done twice (that would panic)
}

// Reads are answered by the state's own goroutine: they get the last state right away, even
// while a transition is still deciding the next one.
func TestGetStateDoesntWaitForATransitionInProgress(t *testing.T) {
	s := newSessionState(OPEN)
	defer s.SetState(CLOSED) // stops the state\'s goroutine
	inTransition, release := make(chan struct{}), make(chan struct{})
	go s.Transition(func(string) string {
		close(inTransition)
		<-release
		return BOUND_TX
	})
	<-inTransition

	got := make(chan string, 1)
	go func() { got <- s.GetState() }()
	select {
	case state := <-got:
		if state != OPEN {
			t.Errorf("GetState() during the transition = %q, want OPEN (the last state)", state)
		}
	case <-time.After(time.Second):
		t.Fatal("GetState() waited for the transition in progress")
	}
	close(release)
	if !eventually(func() bool { return s.GetState() == BOUND_TX }) {
		t.Errorf("GetState() after the transition = %q, want BOUND_TX", s.GetState())
	}
}
