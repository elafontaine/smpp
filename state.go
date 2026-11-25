package smpp

// sessionState holds the session state of an ESME (OPEN, BOUND_TX, ..., CLOSED).
//
// A goroutine owns the state. It hands the current state to every reader that asks, and
// applies transitions one at a time. Readers never wait for a transition: while one is
// deciding the next state, they get the last one. Once the state is CLOSED the goroutine
// stops, and readers get CLOSED from Done.
type sessionState struct {
	reads       chan string     // the owner sends the current state to each reader
	transitions chan transition // requests to change the state, applied one at a time
	done        chan struct{}   // closed when the state becomes CLOSED
}

// transition hands the current state to the caller (lease) and waits for the next one (commit).
type transition struct {
	lease  chan string
	commit chan string
}

func newSessionState(state string) *sessionState {
	s := &sessionState{reads: make(chan string), transitions: make(chan transition), done: make(chan struct{})}
	go s.run(state)
	return s
}

func (s *sessionState) run(state string) {
	transitions := s.transitions
	var commit chan string // the transition in progress, nil when there is none
	for {
		select {
		case s.reads <- state:
		case t := <-transitions: // nil (never ready) while a transition is in progress
			t.lease <- state
			transitions, commit = nil, t.commit
		case next := <-commit:
			state = next
			transitions, commit = s.transitions, nil
			if state == CLOSED {
				close(s.done)
				return
			}
		}
	}
}

// Transition hands f the current state and stores whatever f returns as the next one. It
// returns both, so a caller can tell whether it won a race ("only bind if OPEN"). CLOSED is
// terminal: f is not called and the state does not change.
//
// Other transitions wait while f runs (reads don't), so f must be quick and must not call
// Transition or SetState on the same sessionState. If f panics the state stays unchanged.
func (s *sessionState) Transition(f func(current string) (next string)) (previous, next string) {
	// commit is unbuffered: Transition returns only once the owner has the next state, so a
	// read made after it sees that state.
	t := transition{lease: make(chan string), commit: make(chan string)}
	select {
	case s.transitions <- t:
	case <-s.done:
		return CLOSED, CLOSED
	}
	previous = <-t.lease
	next = previous
	defer func() { t.commit <- next }()
	next = f(previous)
	return previous, next
}

// Done returns a channel that's closed once the state becomes CLOSED, so a select can wait
// for it.
func (s *sessionState) Done() <-chan struct{} {
	return s.done
}

// GetState returns the current state.
func (s *sessionState) GetState() string {
	select {
	case state := <-s.reads:
		return state
	case <-s.done:
		return CLOSED
	}
}

// SetState replaces the current state. It is ignored once the state is CLOSED.
func (s *sessionState) SetState(desired string) {
	s.Transition(func(string) string { return desired })
}
