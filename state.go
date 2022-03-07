package smpp

import (
	"time"
)

type State struct {
	state       string
	askForState chan bool
	setState    chan string
	reportState chan string
	done        chan bool
}

func NewESMEState(state string) *State {
	obj := State{
		state:       state,
		askForState: make(chan bool),
		reportState: make(chan string),
		setState:    make(chan string),
		done:        make(chan bool),
	}
	go obj.stateDispatcher()
	return &obj
}

func (state *State) stateDispatcher() {
	for {
		select {
		case <-state.askForState:
			state.reportState <- state.state
		case msg2 := <-state.setState:
			state.state = msg2
		case <-state.done:
			break
		}
	}
}

func (state *State) getState() string {
	select {
	case state.askForState <- true:
		return <-state.reportState
	case <-time.After(time.Second): // if the channel gets closed, we want to return a state either way
		return CLOSED
	}
}
