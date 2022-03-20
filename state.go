package smpp

import (
	"sync"
	"time"
)

type State struct {
	state       string
	askForState chan bool
	setState    chan string
	reportState chan string
	done        chan bool
	alive       chan bool
	mu          sync.Mutex
}

func NewESMEState(state string) *State {
	obj := State{
		state:       state,
		askForState: make(chan bool),
		reportState: make(chan string),
		setState:    make(chan string),
		done:        make(chan bool),
		alive:       make(chan bool),
	}
	go obj.stateDispatcher()
	return &obj
}

func (state *State) stateDispatcher() {
stateDispatcherLoop:
	for {
		select {
		case <-state.askForState:
			state.reportState <- state.state
		case msg2 := <-state.setState:
			state.state = msg2
		case state.alive <- true:
			continue
		case <-state.done:
			close(state.alive)
			break stateDispatcherLoop
		}
	}
}

func (state *State) getState() string {
	if state.controlLoopStillAlive() {
		select {
		case state.askForState <- true:
			return <-state.reportState
		case <-time.After(time.Second): // if the channel gets closed, we want to return a state either way
		}
	}
	return CLOSED
}

func (state *State) controlLoopStillAlive() bool {
	x, ok := <-state.alive
	if !ok {
		return false
	}
	return x
}

func (state *State) Close() {
	state.mu.Lock()
	if state.controlLoopStillAlive() {
		state.done <- true
	}
	state.mu.Unlock()
}
