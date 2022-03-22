package smpp

import (
	"errors"
	"net"
	"sync/atomic"
	"time"
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
	State           State
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
		State:           *NewESMEState(LISTENING),
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
	for s.State.GetState() != CLOSED {
		select {
		case serverConnectionSocket := <-s.NewConnChan:
			s.createAndAppendNewEsme(serverConnectionSocket)
		case e := <-s.RemoveEsmeChan:
			s._closeAndRemoveEsme(e)
		default:
			time.Sleep(0)
		}
	}
}

func (smsc *SMSC) createAndAppendNewEsme(serverConnectionSocket net.Conn) {
	e := NewEsme(&serverConnectionSocket)
	e.commandFunctions["bind_receiver"] = smsc.handleBindOperation
	e.commandFunctions["bind_transceiver"] = smsc.handleBindOperation
	e.commandFunctions["bind_transmitter"] = smsc.handleBindOperation
	appendNewEsmeToSMSC(smsc, e)
	smsc.NewEsmeChan <- e
}

func appendNewEsmeToSMSC(smsc *SMSC, e *ESME) {
	old_connections := smsc.ESMEs.Load().([]*ESME)
	new_connections := append(old_connections, e)
	smsc.ESMEs.Store(new_connections)
}

func (s *SMSC) closeAndRemoveEsme(e *ESME) {
	s.RemoveEsmeChan <- e
	<-s.RemoveDoneChan
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
	if s.State.GetState() != CLOSED {
		s.State.SetState(CLOSED)
		s.State.done <- true
	}
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
		go s.ensureCleanUpOfEsmes(e)
	}
}

func handleConnection(e *ESME) {
	for e.GetEsmeState() != CLOSED {
		err := handleOperations(e)
		if err != nil {
			InfoSmppLogger.Printf("Issue on Connection: %v\n", err)
		}
		time.Sleep(0)
	}
}

func handleOperations(e *ESME) (formated_error error) {
	receivedPdu, formated_error := e.receivePdu()
	if formated_error != nil {
		return formated_error
	}
	ABindOperation := IsBindOperation(receivedPdu)
	if e.GetEsmeState() == OPEN && !ABindOperation {
		formated_error = handleNonBindedOperations(e, receivedPdu)
	}
	if _, ok := e.commandFunctions[receivedPdu.header.commandId]; ok {
		formated_error = e.commandFunctions[receivedPdu.header.commandId](e, receivedPdu)
	}
	return formated_error
}

func (s *SMSC) ensureCleanUpOfEsmes(e *ESME) {
	go func() {
		defer s.closeAndRemoveEsme(e)
		handleConnection(e)
	}()
}
