package smpp

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
)

const (
	LISTENING = "LISTENING"
)

// SMSC is the server side for the SMPP protocol and is responsible for
// managing connection made to it.  The current implementation allow for
// function registration for configuring SMSC reaction to PDUs.
// However, this is not yet ready for use in Production systems.
type SMSC struct {
	listeningSocket net.Listener
	ESMEs           atomic.Value
	State           *State
	NewConnChan     chan net.Conn
	NewEsmeChan     chan *ESME
	RemoveEsmeChan  chan *ESME
	SystemId        string
	Password        string
	connections     sync.WaitGroup // one per connection still running; only the control loop adds
	loopDone        chan struct{}  // closed once the control loop has returned
}

func NewSMSC(listeningSocket *net.Listener, SystemId string, Password string) (s *SMSC) {
	s = &SMSC{
		listeningSocket: *listeningSocket,
		State:           NewESMEState(LISTENING),
		ESMEs:           atomic.Value{},
		NewConnChan:     make(chan net.Conn),
		NewEsmeChan:     make(chan *ESME),
		RemoveEsmeChan:  make(chan *ESME),
		loopDone:        make(chan struct{}),
		SystemId:        SystemId,
		Password:        Password,
	}
	s.ESMEs.Store([]*ESME{})
	go s.smscControlLoop()
	return s
}

func (smsc *SMSC) acceptNewConnectionFromSMSC() (*ESME, error) {
	serverConnectionSocket, err := smsc.listeningSocket.Accept()
	if err != nil {
		return nil, err
	}
	smsc.NewConnChan <- serverConnectionSocket
	e := <-smsc.NewEsmeChan
	return e, nil
}

// smscControlLoop is the only writer of the ESME list. It never waits on a connection.
func (s *SMSC) smscControlLoop() {
	defer close(s.loopDone)
	for {
		select {
		case serverConnectionSocket := <-s.NewConnChan:
			s.createAndAppendNewEsme(serverConnectionSocket)
		case e := <-s.RemoveEsmeChan:
			s.removeFromList(e)
		case <-s.State.Done():
			return
		}
	}
}

func (smsc *SMSC) createAndAppendNewEsme(serverConnectionSocket net.Conn) {
	e := NewEsme(serverConnectionSocket)
	e.CommandFunctions["bind_receiver"] = smsc.handleBindOperation
	e.CommandFunctions["bind_transceiver"] = smsc.handleBindOperation
	e.CommandFunctions["bind_transmitter"] = smsc.handleBindOperation
	smsc.connections.Add(1) // in the control loop: Shutdown waits only once the loop has returned
	go smsc.watchConnection(e)
	appendNewEsmeToSMSC(smsc, e)
	smsc.NewEsmeChan <- e
}

// watchConnection closes e when its connection ends or when the SMSC closes, whichever comes
// first, then drops it from the list. Closing e waits only for e's own handlers, so a busy
// connection never holds up the others or the SMSC.
func (smsc *SMSC) watchConnection(e *ESME) {
	defer smsc.connections.Done()
	select {
	case <-e.state.Done():
	case <-smsc.State.Done():
	}
	e.Close()
	smsc.removeEsme(e)
}

func appendNewEsmeToSMSC(smsc *SMSC, e *ESME) {
	old_connections := smsc.ESMEs.Load().([]*ESME)
	new_connections := append(old_connections, e)
	smsc.ESMEs.Store(new_connections)
}

// removeEsme has the control loop drop e from the list. Once the SMSC is closed there's no loop
// and nobody keeps the list.
func (s *SMSC) removeEsme(e *ESME) {
	select {
	case s.RemoveEsmeChan <- e:
	case <-s.State.Done():
	}
}

func (s *SMSC) removeFromList(e *ESME) {
	old_connections := s.ESMEs.Load().([]*ESME)
	new_connections := []*ESME{}
	for _, x := range old_connections {
		if x != e {
			new_connections = append(new_connections, x)
		}
	}
	s.ESMEs.Store(new_connections)
}

func (s *SMSC) GetNumberOfConnection() int {
	return len(s.ESMEs.Load().([]*ESME))
}

func (s *SMSC) Start() {
	go s.acceptAllNewConnection()
}

// Close stops the SMSC: it accepts no more connections and every connection is told to close.
// It returns right away, without waiting for them; Shutdown waits.
func (s *SMSC) Close() {
	s.State.SetState(CLOSED)
	s.listeningSocket.Close()
}

// Shutdown closes the SMSC (see Close) and waits until its control loop and every connection
// are gone, or until ctx ends.
func (s *SMSC) Shutdown(ctx context.Context) error {
	s.Close()
	gone := make(chan struct{})
	go func() {
		<-s.loopDone // after this no connection can be added, so Wait can't race an Add
		s.connections.Wait()
		close(gone)
	}()
	select {
	case <-gone:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *SMSC) acceptAllNewConnection() {
	for s.State.GetState() != CLOSED {
		e, err := s.acceptNewConnectionFromSMSC()
		if err != nil {
			InfoSmppLogger.Printf("SMSC wasn't able to accept a new connection: %v", err)
			if errors.Is(err, net.ErrClosed) {
				break //can't get new connection
			}
			continue
		}
		e.StartControlLoop() // answers through the ESME's CommandFunctions; watchConnection cleans up

	}
}
