package smpp

import (
	"sync"
)

type State struct {
	state       string
	setState    chan string
	reportState chan string
	done        chan bool
	alive       chan bool
	mu          sync.Mutex
}

func NewESMEState(state string) *State {
	obj := State{
		state:       state,
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
		case msg2 := <-state.setState:
			state.state = msg2
		case state.reportState <- state.state:
		case state.alive <- true:
			continue
		case <-state.done:
			close(state.alive)
			close(state.reportState)
			break stateDispatcherLoop
		}
	}
}

func (state *State) GetState() string {
	_, ok := <-state.reportState //clear previous one and check if channel is close
	if (ok) {
		return <-state.reportState
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
