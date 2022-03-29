package smpp

import (
	"sync"
)

// Control Loop for managing a ESME state (not a Finite State Machine!).
// The implementation is concurrent safe as SMPP protocol require to know
// which state we're in to take some decisions.
type State struct {
	state       string
	setState    chan string
	reportState chan string
	done        chan bool
	mu          sync.Mutex
}

func NewESMEState(state string) *State {
	obj := State{
		state:       state,
		reportState: make(chan string),
		setState:    make(chan string),
		done:        make(chan bool),
	}
	go obj.stateDispatcher()
	return &obj
}

func (state *State) stateDispatcher() {
stateDispatcherLoop:
	for {
		select {
		case msg2 := <-state.setState:
			state.state = msg2
		case state.reportState <- state.state:
			continue
		case <-state.done:
			close(state.reportState)
			break stateDispatcherLoop
		}
	}
}

func (state *State) GetState() string {
	if state.controlLoopStillAlive() {
		return <-state.reportState
	} 			
	return CLOSED
}

func (state *State) SetState(desired_state string) {
	state.setState <- desired_state
}

func (state *State) controlLoopStillAlive() bool {
	_, ok := <-state.reportState
	return ok
}

func (state *State) Close() {
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.controlLoopStillAlive() {
		state.done <- true
	}
}
