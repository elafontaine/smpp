package smpp

// State holds the session state of an ESME (OPEN, BOUND_TX, ..., CLOSED).
//
// It works like a generator that has to be fed: asking for the current state
// hands it to you, and nobody else can ask until you hand the next one back.
// Transition does both halves in one call, so the state can never be lost and
// a read-decide-write sequence is atomic. There is no goroutine to start or
// stop: the buffered channel itself is the generator.
type State struct {
	token chan string // holds the state; empty while a Transition is running
}

func NewESMEState(state string) *State {
	s := &State{token: make(chan string, 1)}
	s.token <- state
	return s
}

// Transition hands f the current state and stores whatever f returns as the
// next one. It returns both, so a caller can tell whether it won a race
// ("only bind if OPEN"). CLOSED is terminal: f is not called and the state
// does not change.
//
// Other callers wait while f runs, so f must be quick and must not call
// methods on the same State. If f panics the state is handed back unchanged.
func (s *State) Transition(f func(current string) (next string)) (previous, next string) {
	previous = <-s.token
	next = previous
	defer func() { s.token <- next }()
	if previous != CLOSED {
		next = f(previous)
	}
	return previous, next
}

// GetState returns the current state.
func (s *State) GetState() string {
	_, current := s.Transition(func(current string) string { return current })
	return current
}

// SetState replaces the current state. It is ignored once the state is CLOSED.
func (s *State) SetState(desired string) {
	s.Transition(func(string) string { return desired })
}
