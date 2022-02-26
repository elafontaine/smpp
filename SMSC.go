package smpp

import (
	"net"
	"sync/atomic"
)

const (
	LISTENING = "LISTENING"
)

type SMSC struct {
	listeningSocket net.Listener
	ESMEs           atomic.Value
	State           State
}

func NewSMSC(listeningSocket *net.Listener) (s *SMSC) {
	s = &SMSC{
		listeningSocket: *listeningSocket,
		State: *NewESMEState(LISTENING),
		ESMEs: atomic.Value{},
		}
	s.ESMEs.Store([]*ESME{})
	return s
}

func (smsc *SMSC) AcceptNewConnectionFromSMSC() (e *ESME, err error) {
	serverConnectionSocket, err := smsc.listeningSocket.Accept()
	if err != nil {
		return nil, err
	}
	e = NewEsme(&serverConnectionSocket)

	old_connections := smsc.ESMEs.Load().([]*ESME)
	new_connections := append(old_connections, e)
	smsc.ESMEs.Store(new_connections)
	return e, err
}

func (s *SMSC) removeClosedEsmeFromSmsc(e *ESME) {
	e.Close()
	old_connections := s.ESMEs.Load().([]*ESME)
	new_connections := old_connections[:0]
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

func (s *SMSC) Close() {
	for _, conn := range s.ESMEs.Load().([]*ESME) {
		if conn.getEsmeState() != CLOSED {
			conn.Close()
		}
	}
	s.listeningSocket.Close()
	if s.State.getState() != CLOSED {
		s.State.setState <- CLOSED
	}
}
