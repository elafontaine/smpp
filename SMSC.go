package smpp

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
)

const (
	listening = "LISTENING"
)

// SMSC is the server side for the SMPP protocol and is responsible for
// managing connection made to it.  The current implementation allow for
// function registration for configuring SMSC reaction to PDUs.
// However, this is not yet ready for use in Production systems.
type SMSC struct {
	listener    net.Listener
	esmes       atomic.Value // []*ESME, replaced (never changed in place) by the control loop only
	state       *sessionState
	newConns    chan net.Conn
	newEsmes    chan *ESME
	removals    chan *ESME
	SystemId    string
	Password    string
	connections sync.WaitGroup // one per connection still running; only the control loop adds
	loopDone    chan struct{}  // closed once the control loop has returned
}

// NewSMSC serves SMPP on listener: any net.Listener, e.g. from net.Listen ("tcp", "unix") or
// tls.Listen.
func NewSMSC(listener net.Listener, systemId string, password string) *SMSC {
	s := &SMSC{
		listener: listener,
		state:    newSessionState(listening),
		newConns: make(chan net.Conn),
		newEsmes: make(chan *ESME),
		removals: make(chan *ESME),
		loopDone: make(chan struct{}),
		SystemId: systemId,
		Password: password,
	}
	s.esmes.Store([]*ESME{})
	go s.smscControlLoop()
	return s
}

// Connections returns the SMSC's current connections. The slice is a snapshot: it doesn't
// change afterwards.
func (s *SMSC) Connections() []*ESME {
	return s.esmes.Load().([]*ESME)
}

func (smsc *SMSC) acceptNewConnectionFromSMSC() (*ESME, error) {
	serverConnectionSocket, err := smsc.listener.Accept()
	if err != nil {
		return nil, err
	}
	smsc.newConns <- serverConnectionSocket
	e := <-smsc.newEsmes
	return e, nil
}

// smscControlLoop is the only writer of the ESME list. It never waits on a connection.
func (s *SMSC) smscControlLoop() {
	defer close(s.loopDone)
	for {
		select {
		case serverConnectionSocket := <-s.newConns:
			s.createAndAppendNewEsme(serverConnectionSocket)
		case e := <-s.removals:
			s.removeFromList(e)
		case <-s.state.Done():
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
	smsc.newEsmes <- e
}

// watchConnection closes e when its connection ends or when the SMSC closes, whichever comes
// first, then drops it from the list. Closing e waits only for e's own handlers, so a busy
// connection never holds up the others or the SMSC.
func (smsc *SMSC) watchConnection(e *ESME) {
	defer smsc.connections.Done()
	select {
	case <-e.state.Done():
	case <-smsc.state.Done():
	}
	e.Close()
	smsc.removeEsme(e)
}

func appendNewEsmeToSMSC(smsc *SMSC, e *ESME) {
	old_connections := smsc.Connections()
	new_connections := append(old_connections, e)
	smsc.esmes.Store(new_connections)
}

// removeEsme has the control loop drop e from the list. Once the SMSC is closed there's no loop
// and nobody keeps the list.
func (s *SMSC) removeEsme(e *ESME) {
	select {
	case s.removals <- e:
	case <-s.state.Done():
	}
}

func (s *SMSC) removeFromList(e *ESME) {
	old_connections := s.Connections()
	new_connections := []*ESME{}
	for _, x := range old_connections {
		if x != e {
			new_connections = append(new_connections, x)
		}
	}
	s.esmes.Store(new_connections)
}

func (s *SMSC) GetNumberOfConnection() int {
	return len(s.Connections())
}

func (s *SMSC) Start() {
	go s.acceptAllNewConnection()
}

// Close stops the SMSC: it accepts no more connections and every connection is told to close.
// It returns right away, without waiting for them; Shutdown waits.
func (s *SMSC) Close() {
	s.state.SetState(CLOSED)
	s.listener.Close()
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

// acceptAllNewConnection accepts connections until Close closes the listener.
func (s *SMSC) acceptAllNewConnection() {
	for {
		e, err := s.acceptNewConnectionFromSMSC()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			InfoSmppLogger.Printf("SMSC wasn't able to accept a new connection: %v", err)
			continue
		}
		e.StartControlLoop() // answers through the ESME's CommandFunctions; watchConnection cleans up
	}
}
