package smpp

import (
	"errors"
	"net"
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
	RemoveDoneChan  chan bool
	SystemId        string
	Password        string
}

func NewSMSC(listeningSocket *net.Listener, SystemId string, Password string) (s *SMSC) {
	s = &SMSC{
		listeningSocket: *listeningSocket,
		State:           NewESMEState(LISTENING),
		ESMEs:           atomic.Value{},
		NewConnChan:     make(chan net.Conn),
		NewEsmeChan:     make(chan *ESME),
		RemoveEsmeChan:  make(chan *ESME),
		RemoveDoneChan:  make(chan bool),
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

func (s *SMSC) smscControlLoop() {
	for {
		select {
		case serverConnectionSocket := <-s.NewConnChan:
			s.createAndAppendNewEsme(serverConnectionSocket)
		case e := <-s.RemoveEsmeChan:
			s._closeAndRemoveEsme(e)
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
	appendNewEsmeToSMSC(smsc, e)
	smsc.NewEsmeChan <- e
}

func appendNewEsmeToSMSC(smsc *SMSC, e *ESME) {
	old_connections := smsc.ESMEs.Load().([]*ESME)
	new_connections := append(old_connections, e)
	smsc.ESMEs.Store(new_connections)
}

// closeAndRemoveEsme has the control loop close e and drop it from the list. Once the SMSC is
// closed there's no loop and no list left: e is just closed.
func (s *SMSC) closeAndRemoveEsme(e *ESME) {
	select {
	case s.RemoveEsmeChan <- e:
		<-s.RemoveDoneChan
	case <-s.State.Done():
		e.Close()
	}
}

func (s *SMSC) _closeAndRemoveEsme(e *ESME) {
	e.Close()
	old_connections := s.ESMEs.Load().([]*ESME)
	new_connections := []*ESME{}
	for _, x := range old_connections {
		if x != e {
			new_connections = append(new_connections, x)
		}
	}
	s.ESMEs.Store(new_connections)
	s.RemoveDoneChan <- true
}

func (s *SMSC) GetNumberOfConnection() int {
	return len(s.ESMEs.Load().([]*ESME))
}

func (s *SMSC) Start() {
	go s.acceptAllNewConnection()
}

func (s *SMSC) Close() {
	esme_chan := s.getEsmeFromChannel()
	for conn := range esme_chan { //read
		if conn.GetEsmeState() != CLOSED {
			s.closeAndRemoveEsme(conn)
		}
	}
	s.listeningSocket.Close()
	s.State.SetState(CLOSED)
}

func (s *SMSC) getEsmeFromChannel() <-chan *ESME {
	newChannel := make(chan *ESME)
	go func() {
		copy := s.ESMEs.Load().([]*ESME)
		for _, e := range copy {
			newChannel <- e
		}
		close(newChannel)
	}()
	return newChannel
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
		// Run the connection's read loop, which answers through the ESME's CommandFunctions;
		// once the connection is gone, close it and drop it from the list.
		go func() {
			defer s.closeAndRemoveEsme(e)
			e.StartControlLoop()
			e.wg.Wait()
		}()
	}
}
